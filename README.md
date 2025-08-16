This is a project supposed for learning Go with PostgreSQL. 
It aggregaes RSS-feeds

## Setup

In order to run the project you need Postgres and Go installed on your machine.

You can install `gator` CLI using `go install github.com/dimoonchepe/gator` command
You should have a config file named '.gatorconfig.json' in your home directory

It should look something like this:

```
{"db_url":"postgres://postgres:postgres@localhost:5432/gator?sslmode=disable"}
```

## Usage

Login as new user by using `gator login <username>` command

Add preferred RSS feeds by using `gator addfeed <name> <url>` command

Aggregate posts from added feeds by running `gator agg <interval>` command. Is  should be stopped manually by pressing Ctrl+c in the terminal

Read aggregated post titles by running `gator browse <limit>` command. Default limit is 2.

Enjoy!
