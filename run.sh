#!/bin/sh

docker build . -t skiapoden && docker run -it -p 8080:8080 --name skiapoden --rm skiapoden
