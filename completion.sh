_ecms() 
{
    local opts
    COMPREPLY=()
    opts=$(ecms __introspect__ "${COMP_WORDS[@]:1:$COMP_CWORD-1}")

    COMPREPLY=($(compgen -W "${opts}" -- "${COMP_WORDS[1]}"))
}

complete -o default -F _ecms ecms
