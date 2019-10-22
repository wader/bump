#!/bin/bash

# path to bump binary to test
BUMP="$1"

function test_update() {
    local r=0
    local TESTDIR=$(mktemp -d)
    cd "$TESTDIR"

    cat <<EOF > a
bump: name /name: (\d+)/ static:456
EOF

    cat <<EOF > b
name: 123
EOF

    "$BUMP" update a b > stdout 2> stderr 
    local actual_exit_code=$?
    local actual_b=$(cat b)
    local actual_stdout=$(cat stdout)
    local actual_stderr=$(cat stderr)
    local expected_exit_code=0
    local expected_b="name: 456"
    local expected_stdout=""
    local expected_stderr=""

    if [ "$expected_exit_code" != "$actual_exit_code" ] ; then
        echo "exit_code $expected_exit_code got $actual_exit_code"
        r=1
    fi
    if [ "$expected_b" != "$actual_b" ] ; then
        echo "expected_b $expected_b got $actual_b"
        r=1
    fi
    if [ "$expected_stdout" != "$actual_stdout" ] ; then
        echo "stdout $expected_stdout got $actual_stdout"
        r=1
    fi
    if [ "$expected_stderr" != "$actual_stderr" ] ; then
        echo "stderr $expected_stderr got $actual_stderr"
        r=1
    fi

    rm -rf "$TESTDIR"

    return $r
}

function test_diff() {
    local r=0
    local TESTDIR=$(mktemp -d)
    cd "$TESTDIR"

    cat <<EOF > a
bump: name /name: (\d+)/ static:456
EOF

    cat <<EOF > b
name: 123
EOF

    "$BUMP" diff a b > stdout 2> stderr 
    local actual_exit_code=$?

    cat stdout | patch -p0

    local actual_b=$(cat b)
    local actual_stdout=$(cat stdout)
    local actual_stderr=$(cat stderr)
    local expected_exit_code=0
    local expected_b="name: 456"
    local expected_stderr=""

    if [ "$expected_exit_code" != "$actual_exit_code" ] ; then
        echo "exit_code $expected_exit_code got $actual_exit_code"
        r=1
    fi
    if [ "$expected_b" != "$actual_b" ] ; then
        echo "expected_b $expected_b got $actual_b"
        r=1
    fi
    if [ "$expected_stderr" != "$actual_stderr" ] ; then
        echo "stderr $expected_stderr got $actual_stderr"
        r=1
    fi

    rm -rf "$TESTDIR"

    return $r
}


function test_error() {
    local r=0
    local TESTDIR=$(mktemp -d)
    cd "$TESTDIR"

    cat <<EOF > a
bump: name /name: (\d+)/ static:456
EOF

    "$BUMP" update a > stdout 2> stderr 
    local actual_exit_code=$?
    local actual_stdout=$(cat stdout)
    local actual_stderr=$(cat stderr)
    local expected_exit_code=1
    local expected_stdout=""
    local expected_stderr="a:1: name has no current version matches"

    if [ "$expected_exit_code" != "$actual_exit_code" ] ; then
        echo "exit_code $expected_exit_code got $actual_exit_code"
        r=1
    fi
    if [ "$expected_stdout" != "$actual_stdout" ] ; then
        echo "stdout $expected_stdout got $actual_stdout"
        r=1
    fi
    if [ "$expected_stderr" != "$actual_stderr" ] ; then
        echo "stderr $expected_stderr got $actual_stderr"
        r=1
    fi

    rm -rf "$TESTDIR"

    return $r
}

r=0
echo "test_update"
test_update || r=1
echo "test_error"
test_error || r=1
echo "test_diff"
test_diff || r=1

exit $r
