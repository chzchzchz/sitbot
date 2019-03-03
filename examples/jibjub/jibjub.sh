#!/bin/bash

set -eou pipefail

# CONFIGURE THESE
# TARGETNET=irc.whate.ever:6667
# TARGETCHAN="#targetchan"
# TARGETUSER="targetuser"

s1=(a e i o u)
s2=(m n b p d)

function rand_name {
	v1="${s1[$(($RANDOM % 5))]}"
	v2="${s2[$(($RANDOM % 5))]}"
	echo -n j$v1$v2 | sed 's/ap/eb/g'
}

function mk_jj {
	rand_name
	rand_name
}

function mk_word {
	v1="${p1[$(($RANDOM % 4))]}"
	v2="${p2[$(($RANDOM % 4))]}"
	echo -n "J${v1}${v2}"
}

p1=(I O U A)
p2=(BBLE MBLE MMO BLAM)
function mk_phrase {
	echo `mk_word` `mk_word` `mk_word`
}

function mk_bot {
	BOTNAME=`mk_jj`
	BOTPHRASE=`mk_phrase`
	sed "s/BOTNAME/$BOTNAME/g;s/BOTPHRASE/$BOTPHRASE/g;s/TARGETNET/$TARGETNET/g;s/TARGETCHAN/$TARGETCHAN/g;s/TARGETUSER/$TARGETUSER/g;" jj.json
}

for a in `seq 1`; do
	mk_bot | curl -v http://localhost:9991/ -XPOST -d @-
	sleep 1s
done