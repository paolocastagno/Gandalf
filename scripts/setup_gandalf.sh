#!/usr/bin/env bash

set -xe

cnt=0
declare -a SNKS

while getopts "f:T:d:l:t:x:I:A:i:a:r:b:s:S:" opt; do
	echo $opt $OPTARG
	case $opt in
		f) FLG=$OPTARG;;
		T) TME=$OPTARG;;
		d) RND_DTN=$OPTARG;;
		l) LGR=$OPTARG;;
		t) TKN=$OPTARG;;
		x) ATMETO=$OPTARG;;
		I) ID_1=$OPTARG;;
		A) APH_1=$OPTARG;;
		i) ID_2=$OPTARG;;
		a) APH_2=$OPTARG;;
		r) RPSC=$OPTARG;;
		b) BIN=$OPTARG;;
		S) SRVTY=$OPTARG;;
		s) SNKS[$cnt]=$OPTARG;
			echo $OPTARG;
			cnt=$((cnt+1));;
		*) echo 'Error in command line parsing' >&2
			exit 1
	esac
done

SRCDIR=$(pwd)/cfg/${FLG}
DIR=$(pwd)/policy

OPTS=""
OPT=""
if [ $(uname) == "Darwin" ]; then
    OPTS=( -i '' -e ) 
    OPT=( -i '' )
else
    OPTS=( -i ) 
    OPT=( -i )
fi

# # Creating servers' config files
MOD=server
cp ${SRCDIR}/${MOD}/cfg_*.toml $DIR/$MOD/
sed "${OPTS[@]}" "s/SRVST/$SRVTY/g" $DIR/$MOD/cfg_*.toml
# # Creating client's config file
MOD=client
FNM=config_1.json
# player_1
sed "s/BUCKET/measures_game/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TOCKEN/$TKN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_1/${SNKS[0]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_2/${SNKS[1]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/$ID_1/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/$RPSC/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LOGGER/$LGR/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TIME/$TME/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/APPCFG/appconfig_1.toml/g" $DIR/$MOD/$FNM

FNM=appconfig_1.toml
sed "s/DST_1/${SNKS[0]}/g" $SRCDIR/$MOD/app.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/PLID/1/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/A_TO/$ATMETO/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_1/${SNKS[0]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_2/${SNKS[1]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ALPHA/$APH_1/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RND_DRTN/$RND_DTN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/BIN/$BIN/g" $DIR/$MOD/$FNM
# player_2
FNM=config_2.json
sed "s/BUCKET/measures_game/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TOCKEN/$TKN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_1/${SNKS[0]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_2/${SNKS[1]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/$ID_2/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/$RPSC/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LOGGER/$LGR/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TIME/$TME/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/APPCFG/appconfig_2.toml/g" $DIR/$MOD/$FNM

FNM=appconfig_2.toml
sed "s/DST_1/${SNKS[0]}/g" $SRCDIR/$MOD/app.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/PLID/2/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/A_TO/$ATMETO/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_1/${SNKS[0]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST_2/${SNKS[1]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ALPHA/$APH_2/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RND_DRTN/$RND_DTN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/BIN/$BIN/g" $DIR/$MOD/$FNM


# Creating .env file
# InfluxDB
sed "s/TKN/$TKN/g" $SRCDIR/env > ./.env
sed "${OPTS[@]}" "s/BKT/measures_game/g" ./.env
LGRHST="$(echo $LGR | sed 's/"//g')"
sed "${OPTS[@]}" "s/LGR/${LGRHST}/g" ./.env
# Client & Background
EPATH=$( echo $DIR/client/appconfig_1.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/PLR_1_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/appconfig_2.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/PLR_2_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/config_1.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/PLR_1_C/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/config_2.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/PLR_2_C/${EPATH}/g" ./.env
# Server
EPATH=$( echo $DIR/server/cfg_trn.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/SRV_TRN_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/server/cfg_md.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/SRV_MD_A/${EPATH}/g" ./.env