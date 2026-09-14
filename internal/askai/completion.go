package askai

// BashCompletion returns completion definitions for the main and helper commands.
func BashCompletion() string {
	return `_ask_ai_providers="chatgpt claude gemini kimi custom"

_ask_ai_provider() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  COMPREPLY=($(compgen -W "${_ask_ai_providers}" -- "${cur}"))
}

_ask_ai() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=($(compgen -W "provider completion -config -from-clipboard -version" -- "${cur}"))
    return
  fi

  case "${COMP_WORDS[1]}" in
    provider)
      if [[ ${COMP_CWORD} -eq 2 ]]; then
        COMPREPLY=($(compgen -W "${_ask_ai_providers}" -- "${cur}"))
      fi
      ;;
    completion)
      COMPREPLY=($(compgen -W "bash" -- "${cur}"))
      ;;
  esac
}

complete -F _ask_ai ask-ai
complete -F _ask_ai_provider ask-ai-provider
`
}
