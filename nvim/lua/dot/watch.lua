-- Watch functionality for Graphviz DOT files.
-- Starts dotx watch server and opens browser for live SVG preview.

local M = {}

local job_id = nil
local watch_buf = nil
local url = nil

--- Stop the watch server if running
local function stop()
  if job_id then
    vim.fn.jobstop(job_id)
    job_id = nil
    url = nil
    watch_buf = nil
  end
end

--- Start the watch server for the given file
---@param file string absolute path to the DOT file
---@return boolean success
local function start(file)
  if vim.fn.executable('dotx') ~= 1 then
    vim.notify('dotx not found in PATH. Install with: go install github.com/teleivo/dot/cmd/dotx@latest', vim.log.levels.ERROR)
    return false
  end

  watch_buf = vim.api.nvim_get_current_buf()

  job_id = vim.fn.jobstart({ 'dotx', 'watch', file }, {
    on_stdout = function(_, data)
      if url then
        return -- already got the URL
      end
      for _, line in ipairs(data) do
        local match = line:match('(http://[%d%.]+:%d+)')
        if match then
          url = match
          vim.ui.open(url)
          vim.notify('dotx watch: ' .. url)
          return
        end
      end
    end,
    on_stderr = function(_, data)
      for _, line in ipairs(data) do
        if line ~= '' then
          vim.notify('dotx watch: ' .. line, vim.log.levels.WARN)
        end
      end
    end,
    on_exit = function(_, code)
      -- 143 = 128 + SIGTERM(15), normal termination via jobstop
      if code ~= 0 and code ~= 143 then
        vim.notify('dotx watch exited with code ' .. code, vim.log.levels.ERROR)
      end
      job_id = nil
      url = nil
      watch_buf = nil
    end,
    stdout_buffered = false,
    stderr_buffered = false,
  })

  if job_id <= 0 then
    vim.notify('Failed to start dotx watch', vim.log.levels.ERROR)
    job_id = nil
    return false
  end

  return true
end

--- Toggle watch server for current buffer
function M.toggle()
  local current_buf = vim.api.nvim_get_current_buf()

  -- If watching current buffer, stop
  if job_id and watch_buf == current_buf then
    stop()
    vim.notify('dotx watch stopped')
    return
  end

  -- If watching different buffer, stop first
  if job_id then
    stop()
  end

  local file = vim.api.nvim_buf_get_name(current_buf)
  if file == '' then
    vim.notify('Buffer has no file', vim.log.levels.ERROR)
    return
  end

  start(file)
end

--- Check if watch is active for current buffer
---@return boolean
function M.is_active()
  return job_id ~= nil and watch_buf == vim.api.nvim_get_current_buf()
end

-- Stop on VimLeavePre
vim.api.nvim_create_autocmd('VimLeavePre', {
  group = vim.api.nvim_create_augroup('DotWatch', { clear = true }),
  callback = stop,
})

-- Stop when watched buffer is deleted
vim.api.nvim_create_autocmd('BufDelete', {
  group = vim.api.nvim_create_augroup('DotWatchBufDelete', { clear = true }),
  callback = function(ev)
    if watch_buf and ev.buf == watch_buf then
      stop()
    end
  end,
})

return M
