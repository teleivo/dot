-- Neovim plugin for Graphviz DOT language development.
-- See https://github.com/teleivo/dot

local M = {}

M.inspect = require('dot.inspect')
M.lsp = require('dot.lsp')
M.watch = require('dot.watch')

return M
