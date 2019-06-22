#!/bin/sh

# test against Heroku (inline JSON)
curl -X POST https://skiapoden.herokuapp.com/firstlink -d '{ "language": "ru", "article": "Пиво" }'

# test against localhost (JSON from a file)
curl -X POST localhost:8080/firstlink -d @firstlink.json
