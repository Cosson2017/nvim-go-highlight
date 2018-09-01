""""""""""""""""""""""""""""""""""""""""""
"    LICENSE: 
"     Author: 
"    Version: 
" CreateTime: 2018-08-24 10:51:17
" LastUpdate: 2018-08-24 10:51:17
"       Desc: 
""""""""""""""""""""""""""""""""""""""""""


if exists('g:loaded_hello')
  finish
endif
let g:loaded_hello = 1

function! s:RequireGoHighlight(host) abort
  " 'hello' is the binary created by compiling the program above.
  return jobstart(['gohighlight'], {'rpc': v:true})
endfunction

call remote#host#Register('gohighlight', 'x', function('s:RequireGoHighlight'))
" The following lines are generated by running the program
" command line flag --manifest hello
call remote#host#RegisterPlugin('hello', '0', [
    \ {'type': 'function', 'name': 'Serve', 'sync': 1, 'opts': {}},
    \ ])
