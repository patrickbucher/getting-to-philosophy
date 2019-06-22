#!/bin/sh

# test against Heroku
curl -X POST https://skiapoden.herokuapp.com/csv --data-binary @tests-de.csv >report-de.csv

# test against localhost
curl -X POST localhost:8080/csv --data-binary @tests.csv >report.csv
