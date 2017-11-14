
function red {
  echo "\033[0;31m${1}\033[0m"
}
function green {
  echo "\033[0;32m${1}\033[0m"
}

function test_assert {
  if [ "$1" != "$2" ]
  then
    fail "$3" "$1" "$2"
  else
    success "$3"
  fi
}

function test_contains {
  if [[ "$2" == *"$1"* ]]
  then
    success "$3"
  else
    fail "$3" "$1" "$2"
  fi
}

function fail {
    echo -e "[$(datef) $ME]: ${1} $(red FAILED) \n Expected: ${2} \n GOT: ${3}"
    exit 1
}

function success {
    echo -e "[$(datef) $ME]: ${1} $(green OK)"
}


function datef
{
  date +'%Y-%m-%dT%H:%M:%S%z'
}

log() {
  echo "[$(datef) $ME]: $@"
}