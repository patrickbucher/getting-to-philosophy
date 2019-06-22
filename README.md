# firstlink

Microservice to find the first article link of a Wikipedia article

## Run

Start using Docker:

```sh
$ docker build . -t skiapoden
$ docker run -it -p 8080:8080 --name skiapoden --rm skiapoden
```

Or, easier, use the script:

```sh
$ ./run.sh
```

Start using Go:

```sh
$ PORT=8080 go run firstlink.go
```

## Use

Go to [localhost:8080](http://localhost:8080), if you run the application locally, or use the Heroku deployment on [skiapoden.herokuapp.com](https://skiapoden.herokuapp.com).
