# sitbot

```
   __________.
  /_/-----/_/|   __
  ( ( ' ' ( (| /'--'\
  (_( ' ' (_(|/.    .\
  / /=====/ /|  '||'
 /_//____/_/ |   ||
(o|:.....|o) |   ||
|_|:_____|_|/'  _||_
 '        '    /____\
```

good bot

## Running

Launch `sitbot` to listen on `localhost:12345`:
```sh
go get github.com/chzchzchz/sitbot
go get github.com/chzchzchz/sitbot/cmd/sitbox
sitbot localhost:12345 &
```

## Bots

### Profiles

A bot profile has connection information and regular expression pattern matching rules to control script activation. One sitbot process can manage multiple profiles connectiong to multiple servers. Post a JSON-encoded profile to the sitbot server to launch a new bot; see [profile.json](profile.json) for an example.

### Management

Connect to an IRC network by posting a bot profile to sitbot:
```sh
curl localhost:12345 -XPOST -d@profile.json
```
Reposting a bot profile will update the bot's pattern matching rules.

Configure a control panel page:
``sh
curl localhost:12345/tmpl -XPOST -d@bot.tmpl
```
Any connected bots will be viewable via the generated HTML at `http://localhost:12345/`.

Disconnect a bot by deleting its profile identifier:
```sh
curl localhost:12345/bot/mainbot -XDELETE
```

## Bouncer

sitbot can listen on ports and relay IRC messages between the bot and another IRC client. By connecting through the bouncer, a client sees the bot's IRC session and can issue IRC commands through bot user.

Create a bouncer:
```sh
curl localhost:12345/bouncer/mainbot -XPOST -d localhost:7777
irssi -c localhost -p 7777
```

