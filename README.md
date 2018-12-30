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
sitbot localhost:12345 &
```

Connect to an IRC network by posting a bot profile to sitbot:
```sh
curl localhost:12345 -XPOST -d@profile.json
```
Reposting a bot profile will update the bot's pattern matching rules.

Disconnect a bot by deleting its profile identifier:
```sh
curl localhost:12345/bot/mainbot -XDELETE
```

Create a bouncer:
```sh
curl localhost:12345/bouncer/mainbot -XPOST -d localhost:7777
```