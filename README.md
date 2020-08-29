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
docker build -t restaurant-tracker .
docker run --name restaurant-tracker-0 -p 3002:8888 -v /var/db/restaurant-tracker:/var/db/restaurant-tracker -d -e DBPATH --log-opt max-size=100m --log-opt max-file=12 --log-opt compress=true restaurant-tracker
```

The DBPATH environment variable should be the full path on your host to the database. You can follow logs by running `docker logs -f restaurant-tracker-0`.

## Running for development

1. Pull down all the dependencies.
    ```
    cd cmd/api-server
    go build ./..
    ```
2. The build and run the api-server
    ```
    cd cmd/api-server
    go build .
    ./api-server -db ../../database/your-sqlite3.db -v
    ```