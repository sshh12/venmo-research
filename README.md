# Venmo Research

> The code used in the paper *Contact Tracing With Venmo* as a part of UT Austin's [Computational Media Lab](http://www.computationalmedialab.com/).

![methods](https://github.com/sshh12/venmo-research/blob/master/notebooks/fig_methods.png)

## Usage & Replication

### Database Setup

You'll need a fresh [postgres](https://www.postgresql.org/download/) database hosted on a server with at least ~100GB of free storage. If you have the disk space and reasonable hardware could also just download and run postgres locally. I also heavily recommend the use of [pgAdmin](https://www.pgadmin.org/) for debugging and exploring the database.

You'll need to add have following environment variables when running all the commands below: `POSTGRES_PASS`, `POSTGRES_ADDR`, `POSTGRES_USER`, `POSTGRES_DB`. For example:

```
export POSTGRES_PASS=password
export POSTGRES_ADDR=127.0.0.1:5432
export POSTGRES_USER=postgres
export POSTGRES_DB=venmo
```

### Download Research Code

Download and extract the latest binaries from [releases](https://github.com/sshh12/venmo-research/releases).

If you're familar with Go you could also clone this repo and `go run` things. 

### Venmo Collection

1. Create a Venmo account
2. Use your Venmo login to generate an API key with [scripts/login.py](https://github.com/sshh12/venmo-research/blob/master/scripts/login.py). This only has to be done once as the API key does not expire.
3. Collect data

##### Randomly scrape transactions by user
```
./scrape -mode transactions -token <your API key here> -random
```

##### Scrape transactions of user with an ID between 0 and 95000000 using 5 parallel workers.
```
./scrape -mode transactions -token <your API key here> -start_id 0 -end_id 95000000 -workers 5
```

##### As machine 2 of 10 (0-indexed), scrape transactions of users with an ID between 0 and 95000000 using 5 parallel workers.
```
./scrape -mode transactions -token <your API key here> -start_id 0 -end_id 95000000 -workers 5 -shard_idx 2 -shard_cnt 10
```

##### Continously scrape the latest transactions from `https://venmo.com/api/v5/public`.
```
./scrape -mode transactions2 -token <your API key here>
```

##### View help
```
./scrape -h
```

### Name Search (finding social media profiles)

##### Randomly sample Venmo users from database and look them up on Bing, DuckDuckGo, and PeekYou.

```
./scrape -mode namesearch -workers 1
```

### Geotag Extraction (scraping Facebook)

1. Create a Facebook account (the account must be created with a phone number to avoid being blocked)
2. Install [Chrome](https://www.google.com/chrome/)
3. Download the [chromedriver](https://chromedriver.chromium.org/downloads) and as well as the latest [selenium server](https://www.selenium.dev/downloads/)
4. Collect data

##### Randomly sample users with PeekYou matches and extract geotags

```
./scrape -mode peekyoulocs -fb_user <facebook phone number> -fb_pass <facebook password> -sel_driver chromedriver -sel_headless -workers 3
```

### Analysis & Visualization

1. Open a [jupyter notebook](https://github.com/sshh12/venmo-research/tree/master/notebooks) in this repo
2. Pip install necessary dependencies
3. Edit the `connect()` function to match the parameters for your database
4. Run the notebook cells in order

## Our Dataset

Creating our dataset took several months and with several API changes Venmo collection may no longer be possible at this scale (135M transactions, 22.1M users). Open an issue here or contact us if you would like to receive a copy of our dataset (note: we'll need to verify your use case and intentions before hand, additional restrictions may apply).

Use (with parameters adjusted for your postgres installation) to replicate the database used when running our notebooks:
```
$ pg_restore --host "127.0.0.1" --port "5432" --username "postgres" --no-password --dbname "venmo" --verbose "dataset.sql"
```

## [TACC](https://www.tacc.utexas.edu/) Suggestions

TACC can be a huge pain compared to any cloud provider but it can be useful as a free (for us at UT) compute resource. Personally, I only used it for jobs running with the `transactions` and `namesearch` mode. You can use [scripts/scrape.tacc.job](https://github.com/sshh12/venmo-research/blob/master/scripts/scrape.tacc.job) as a template for doing this. Keep in mind that you'll need to download and extract the [latest release](https://github.com/sshh12/venmo-research/releases), update the environment variables (see placeholders in the script), and run `$ sbatch scrape.tacc.job` while on a `stampede2.tacc.utexas.edu` login node.

It would be extremely useful to run postgres directly on TACC, but running a database as a job is pretty weird (I contacted them and that's only way of doing it now) as it will only run for fix amount of time (e.g. 6 hours) before shutting down and you'll have to wait for the job queue before it even starts. If you do want to still try this, I've left some snippets below that may be useful.

```bash
# after starting an interactive job w/idev
# use docker (TACC uses docker alt called singularity) to run postgres server
module load tacc-singularity
singularity pull docker://postgres
SINGULARITYENV_POSTGRES_PASSWORD=pgpass SINGULARITYENV_PGDATA=$SCRATCH/pgdata singularity run --cleanenv --bind $SCRATCH:/var postgres_latest.sif

# portforwarding with ssh magic (copied from VNC demo script), you could maybe ngrok tcp 5432 instead (?)
NODE_HOSTNAME=`hostname -s`
for i in `seq 4`; do
    ssh -q -f -g -N -R 15426:$NODE_HOSTNAME:15426 login$i
done
ssh -f -N -L 15426:stampede2.tacc.utexas.edu:15426 <your username>@stampede2.tacc.utexas.edu
```