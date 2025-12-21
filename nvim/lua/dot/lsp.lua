-- LSP client configuration for dotx lsp.
-- Provides diagnostics for Graphviz DOT files.

local M = {}

local client_id = nil

--- Get the LSP client configuration
---@return vim.lsp.ClientConfig
local function get_config()
  return {
    name = 'dotlsp',
    cmd = { 'dotx', 'lsp' },
    root_dir = vim.fn.getcwd(),
    filetypes = { 'dot' },
  }
end

--- Start the LSP client for the current buffer
---@return integer|nil client_id
function M.start()
  if client_id and vim.lsp.get_client_by_id(client_id) then
    vim.lsp.buf_attach_client(0, client_id)
    return client_id
  end

  client_id = vim.lsp.start(get_config())
  return client_id
end

--- Setup the LSP to auto-attach to DOT files
function M.setup()
  vim.api.nvim_create_autocmd('FileType', {
    group = vim.api.nvim_create_augroup('DotLsp', { clear = true }),
    pattern = 'dot',
    callback = function()
      M.start()
    end,
  })
end

return M
