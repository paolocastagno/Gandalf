#!/usr/bin/env bash

set -xe

cnt_sp=0
declare -a SNKSP
cnt_np=0
declare -a NXTP
cnt_sh=0
declare -a SNKSH
cnt_nh=0
declare -a NXTH

while getopts "f:T:j:i:l:t:I:r:d:R:D:s:n:S:N" opt; do
	case $opt in
		f) FLG=$OPTARG;;
		T) TYPE=$OPTARG;;
		j) j=$OPTARG;;
		i) i=$OPTARG;;
		l) LGR=$OPTARG;;
		t) TKN=$OPTARG;;
		I) ID=$OPTARG;;
		r) RPSC=$OPTARG;;
		d) DSTC=$OPTARG;;
		R) RPSB=$OPTARG;;
		D) DSTB=$OPTARG;;
		s) SNKSP[$cnt_sp]=$OPTARG;
			cnt_sp=$((cnt_sp+1));;
		n) NXTP[$cnt_np]=$OPTARG;
			cnt_np=$((cnt_np+1));;
		S) SNKSH[$cnt_sh]=$OPTARG;
			cnt_sp=$((cnt_sh+1));;
		N) NXTH[$cnt_nh]=$OPTARG;
			cnt_nh=$((cnt_nh+1));;
		*) echo 'Error in command line parsing' >&2
			exit 1
	esac
done

echo "Proxy sink: ${SNKSP[@]}"
echo "Proxy hop: ${NXTP[@]}"
echo "Hop sink: ${SNKSH[@]}"
echo "Hop hop: ${NXTH[@]}"

# Global parameters
# FLG=${1:?flag missing}
# I=${2:?index missing}
# LGR=${3:?loger missing}
# TKN=${4:?tocken missing}
# Client's parameter
# ID=${5:?cli ID missing}
# RPSC=${6:?cli requests per second missing}
# DSTC=${7:?cli destinations missing}
# Background's parameters
# RPSB=${8:?bg requests per second missing}
# DSTB=${9:?bg destinations missing}
# Proxy's parameter
# SNKSP=${10:?proxy sinks missing}
# NXTP=${11:?proxy servers missing}
# Hop's parameter
# SNKSH=${12:?hop sinks missing}
# NXTH=${13:?proxy servers missing}

SRCDIR=$(pwd)/cfg/${FLG}
DIR=$(pwd)/policy

SRC=$(pwd)/cfg/${FLG}/routing/psl_${TYPE}
PSL=($(cat $SRC))
SRC=$(pwd)/cfg/${FLG}/routing/psm_${TYPE}
PSM=($(cat $SRC))
SRC=$(pwd)/cfg/${FLG}/routing/psh_${TYPE}
PSH=($(cat $SRC))

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
# # Creating client's config file
MOD=client
FNM=config.json
sed "s/BUCKET/measures_${FLG}_${TYPE}_${j}_${i}/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TOCKEN/$TKN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DESTINATIONS/$DSTC/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/$ID/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/$RPSC/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LOGGER/$LGR/g" $DIR/$MOD/$FNM
# Creating client application configuration
FNM=app.toml
sed "s/DESTINATIONS/$DSTC/g" $SRCDIR/$MOD/$FNM > $DIR/$MOD/$FNM
# # Creating background's config file
FNM=config-bg.json
sed "s/DESTINATIONS/$DSTB/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/${ID}-bg/g" $DIR/$MOD/$FNM
rpsb=$(bc -l <<< $RPSB*$((i-2))) # $(($RPSB*$((i-1)) | bc -l))
sed "${OPTS[@]}" "s/RPS/${rpsb}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/toml\",/toml\"/g" $DIR/$MOD/$FNM
sed "${OPT[@]}" '9,17d' $DIR/$MOD/$FNM
# Creating routing's config file
MOD=routing
FNM=config.toml
sed "s/PSL/${PSL[$((i-1))]}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/PSM/${PSM[$((i-1))]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/PSH/${PSH[$((i-1))]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST1/${SNKSP[0]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST2/${SNKSP[1]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DST3/${SNKSP[2]}/g" $DIR/$MOD/$FNM
# Creating hps's config file
MOD=hop
FNM=config.toml
sed "s/DST/${SNKSH[0]}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM

# Creating .env file
# InfluxDB
sed "s/TKN/$TKN/g" $SRCDIR/env > ./.env
sed "${OPTS[@]}" "s/BKT/measures_${FLG}_${TYPE}_${j}_${i}/g" ./.env
LGRHST="$(echo $LGR | sed 's/"//g')"
sed "${OPTS[@]}" "s/LGR/${LGRHST}/g" ./.env
# Client & Background
EPATH=$( echo $DIR/client/app.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_A/${EPATH}/g" ./.env
sed  "${OPTS[@]}" "s/BG_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/config.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_C/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/config-bg.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/BG_C/${EPATH}/g" ./.env
# Proxy
EPATH=$( echo $DIR/routing/$FNM | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/PXY_C/${EPATH}/g" ./.env
# SNKPXY="$(echo $SNKSP | sed 's/"//g' | sed 's/,//g')"
# SNKPXY="$(echo $SNKPXY | sed 's/,//g')"
# sed "${OPTS[@]}" "s/PXY_S/${SNKPXY}/g" ./.env
snksp=$(echo "${SNKSP[@]}")
sed "${OPTS[@]}" "s/PXY_S/${snksp}/g" ./.env
nxtp=$(echo "${NXTP[@]}")
sed "${OPTS[@]}" "s/PXY_D/${nxtp}/g" ./.env
# Hop
EPATH=$( echo $DIR/hop/$FNM | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/HP_C/${EPATH}/g" ./.env
# SNKHOP="$(echo $SNKSP | sed 's/"//g' | sed 's/,//g')"
# SNKHOP="$(echo $SNKHOP | sed 's/,//g')"
# sed "${OPTS[@]}" "s/PXY_S/${SNKHOP}/g" ./.env
snksh=$(echo "${SNKSH[@]}")
sed "${OPTS[@]}" "s/HP_S/${snksh}/g" ./.env
nxth=$(echo "${NXTH[@]}")
sed "${OPTS[@]}" "s/HP_D/${snksh}/g" ./.env
# Server
EPATH=$( echo $DIR/server/cfg_trn.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/SRV_TRN_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/server/cfg_md0.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/SRV_MD0_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/server/cfg_md1.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/SRV_MD1_A/${EPATH}/g" ./.env
