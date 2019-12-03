#!/usr/bin/env bats

@test "runs a command with -c" {
  result="$(wash -c 'echo hello')"
  [ "$result" = hello ]
}

@test "runs a command with input from stdin" {
  result="$(echo 'echo hello' | wash)"
  [ "$result" = hello ]
}
