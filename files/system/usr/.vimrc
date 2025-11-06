" ~/.vimrc - Vim configuration file

" Basic settings
set nocompatible              " Use Vim settings, not Vi
syntax on                     " Enable syntax highlighting
filetype plugin indent on     " Enable filetype detection

" Display settings
set number                    " Show line numbers
set ruler                     " Show cursor position
set showcmd                   " Show command in bottom bar
set showmatch                 " Highlight matching brackets
set cursorline                " Highlight current line

" Indentation
set tabstop=4                 " Number of spaces per tab
set softtabstop=4             " Number of spaces per tab when editing
set shiftwidth=4              " Number of spaces for autoindent
set expandtab                 " Convert tabs to spaces
set autoindent                " Auto-indent new lines

" Search settings
set incsearch                 " Search as characters are entered
set hlsearch                  " Highlight search matches
set ignorecase                " Ignore case when searching
set smartcase                 " Override ignorecase if search has uppercase

" Behavior
set backspace=indent,eol,start " Make backspace work as expected
set wildmenu                  " Visual autocomplete for command menu
set laststatus=2              " Always show status line
set encoding=utf-8            " Use UTF-8 encoding

" Disable backup files
set nobackup
set nowritebackup
set noswapfile

" Performance
set lazyredraw                " Don't redraw while executing macros

" Color scheme (uncomment if you have a preferred theme)
" colorscheme desert
