#!/bin/sh

for i in {1..4}
do
  echo "Command no. $i"
  sleep 0.5
done

>&2 echo "Error 1"

for i in {5..7}
do
  echo "Command no. $i"
  sleep 0.5
done

>&2 echo "Error 2"

for i in {8..10}
do
  echo "Command no. $i"
  sleep 0.5
done