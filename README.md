# wows-recruiting-bot

Discord bot monitoring clan exits to spot potential recruits.

This bot will scan a user defined list of clans at regular interval. 
Whenever a player leave a monitored clan, it will send a Discord message if the player match some minimum criterias:
* minimum Win Rate
* minimum number of Battles
* minimum number of T10 ships
* maximum time since last battle

# Usage

## Commands

The bot provides the following slash commands:
* **/wows-recruit-set-filter**: Set minimum filters for players (min WR, min battles, etc)
* **/wows-recruit-get-filter**: Display the current filter
* **/wows-recruit-replace-clans**: Set the list of monitored clans, takes a CSV file as input, the first column must be the clan tag, other columns are ignored, be aware it replaces the whole list
* **/wows-recruit-list-clans**: List the currently monitored clans, returns a CSV file
* **/wows-recruit-add-clan**: Add a single clan to the monitored list
* **/wows-recruit-remove-clan**: Remove a single clan from the monitored list
* **/wows-recruit-remove-test**: Simple test triggering a fake "player left" message 

## How to use

To use this bot, you first need to invite it to a channel (tips: a dedicated channel is preferable).

Then you need to set the filter using the **/wows-recruit-set-filter** command.

And finally, you need to provide the list of monitored clans, either by adding them individually through **/wows-recruit-add-clan** or in batch through **/wows-recruit-replace-clans**.

Once done, you should start receiving messages within two hours (monitored clans are scanned every 2 hours). 

# Build

## Requirement

You need to have `golang` > 1.19 and `make` installed

## Building

To build wows-recruiting-bot, run:

```shell
make
```

# Running

To run this bot, you need:
* a WoWs API key
* a Discord bot token & Application

## WoWs API key

To get a WoWs API key, please refer to [the Wargaming Developer Documentation](https://developers.wargaming.net/documentation/guide/getting-started/)

## Discord Bot Token

To get a Discord bot token, please refer to [the Discord Developer Portal](https://discord.com/developers/docs/getting-started)

## Running the bot

To run the bot, first, export the following variables:

```bash
export WOWS_DISCORD_TOKEN=MTB....
export WOWS_WOWSAPIKEY=2b4...........
export WOWS_REALM=eu
export WOWS_DEBUG=false
```
Then, launch the bot:

```bash
./wows-recruiting-bot
```

If you run it for the first time, the bot needs to populate the DB.
This process recovers all the clans on a given realm/server and can take several hours.
Be patient.

The bot data are stored in the `wows-recruiting-bot.db` sqlite DB.

## Data updates frequency

Monitored clans are updated every **2 hours**.

All clans (and theirplayers) are updated **once a week**.

