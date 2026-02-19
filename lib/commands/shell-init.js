function shellInit() {
  const script = `wt() {
  if [ "$1" = "cd" ]; then
    shift
    local dir
    dir="$(command wt cd "$@")"
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir"
    else
      return 1
    fi
  elif [ "$1" = "create" ] || [ "$1" = "pr" ]; then
    local name="$2"
    command wt "$@" || return $?
    local dir
    dir="$(command wt cd "$name")"
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir"
    fi
  else
    command wt "$@"
  fi
}`;
  console.log(script);
}

module.exports = shellInit;
