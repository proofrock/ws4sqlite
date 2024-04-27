#!/bin/bash

URL="http://localhost:12321/test_ws4sql"
REQUESTS=100000

cd "$(dirname "$0")"

rm -f environment/*.db*
rm -f ws4sql*

pkill -x ws4sql

cd ..
make build-nostatic
cp bin/ws4sql profiler/
cd profiler

./ws4sql --db environment/test_ws4sql.db &

javac Profile.java

sleep 1

echo -n "Elapsed seconds: "
java -cp ./ Profile $REQUESTS $URL $REQ

rm Profile.class

pkill -x ws4sql

rm -f ws4sql*
rm -f environment/*.db*
