# Restaurant Tracker

## What

This app that will help track the restaurants you've already visited or are planning on visiting. You can share access with your partner, friends, or family so they have access too.

## Why

I was using a shared Google Sheet before to do this. But I wanted a more mobile friendly UI. I also wanted better input validation, easier per visit tracking, and Google Maps integration. Finally, I wanted privacy, specifically the ability to run the app on my own server.

## Creating the database

1. [Download and install sqlite3](https://sqlite.org/download.html) for your operating system.
    ```
    # On Debian/Ubuntu
    apt install sqlite3
    ```
2. Then create the database
    ```
    cat database/create-db.sql | sqlite3 database/your-sqlite3.db
    ```
3. You can inspect your database using `sqlite3 database/your-sqlite3.db` or [DB Browser for Sqlite](https://sqlitebrowser.org/).

## Running with Docker

```
export DBPATH=/var/db/restaurant-tracker/your-sqlite3.db
export SECRETKEY=your-secret-key-for-authentication-cookies
export GMAPSKEY=your-google-maps-api-key
export CSRFKEY=your-csrf-key
docker build -t restaurant-tracker .
docker run --name restaurant-tracker-1 -p 3002:8080 -v /var/db/restaurant-tracker:/var/db/restaurant-tracker -d -e DBPATH -e SECRETKEY -e GMAPSKEY -e CSRFKEY --log-opt max-size=100m --log-opt max-file=12 --log-opt compress=true restaurant-tracker
```

The DBPATH environment variable should be the path to your database in the container. The first part of the `-v` option is the directory of your database on your host. The second part is the directory in your container where you want to have your database. In other words, it is the directory of $DBPATH. 

You can follow logs by running `docker logs -f restaurant-tracker-1` when the container is running.

## Running for development

1. Pull down all the dependencies.
    ```
    cd cmd/api-server
    go get ./...
    ```
2. Then build the web-server
    ```
    cd cmd/web-server
    go build . -o ../../web-server
    ```
3. Set the required environment variables
    ```
    export SECRETKEY=your-secret-key-for-authentication-cookies
    export GMAPSKEY=your-google-maps-api-key
    export CSRFKEY=your-csrf-key
    ```
3. Then run it from the root directory of this project
    ```
    cd ../../
    ./web-server -db ./database/dev-kelvin-4.db -v -csrf $CSRFKEY
    ```