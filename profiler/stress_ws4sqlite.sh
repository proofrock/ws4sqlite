#!/bin/bash

URL="http://localhost:12321/test_ws4sqlite"
REQUESTS=100000

cd "$(dirname "$0")"

rm -f environment/*.db*
rm -f ws4sqlite*

pkill -x ws4sqlite

cd ..
make build-nostatic
cp bin/ws4sqlite profiler/
cd profiler

./ws4sqlite --db environment/test_ws4sqlite.db &

javac Profile.java

sleep 1

echo -n "Elapsed seconds: "
java -cp ./ Profile $REQUESTS $URL $REQ

rm Profile.class

pkill -x ws4sqlite

rm -f ws4sqlite*
rm -f environment/*.db*
