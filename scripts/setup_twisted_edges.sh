#!/usr/bin/env bash

set -xe


while getopts "f:j:i:l:T:s:p:q:S:P:Q:r:u:d:U:D:v:e:w:g:" opt; do
	case $opt in
		f) FLG=$OPTARG;;
		j) j=$OPTARG;;
		i) i=$OPTARG;;
		l) LGR=$OPTARG;;
		T) TKN=$OPTARG;;
		s) SRV_MD=$OPTARG;;
		# p) PT_MD_CLI=$OPTARG;;
		p) PT_MD_DEL=$OPTARG;;
		q) PT_MD_SRV=$OPTARG;;
		S) SRV_TRN=$OPTARG;;
		# P) PT_TRN_CLI=$OPTARG;;
		P) PT_TRN_DEL=$OPTARG;;
		Q) PT_TRN_SRV=$OPTARG;;
		r) RPS=$OPTARG;;
		u) LTUPH_MD=$OPTARG;;
		d) LTDNH_MD=$OPTARG;;
		U) LTUPH_TRN=$OPTARG;;
		D) LTDNH_TRN=$OPTARG;;
		v) LTUPD_MD=$OPTARG;;
		e) LTDND_MD=$OPTARG;;
		V) LTUPD_TRN=$OPTARG;;
		E) LTDND_TRN=$OPTARG;;
		w) LTUP_UPF=$OPTARG;; 
		g) LTDN_UPF=$OPTARG;;
		*) echo 'Error in command line parsing' >&2
			exit 1
	esac
done

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
LOC=md
FNM=cfg_${LOC}.toml
sed "s/DESTINATION/${SRV_TRN}:${PT_TRN_SRV}/g" $SRCDIR/$MOD/cfg.toml > $DIR/$MOD/$FNM
LOC=trn
FNM=cfg_${LOC}.toml
sed "s/DESTINATION/${SRV_MD}:${PT_MD_SRV}/g" $SRCDIR/$MOD/cfg.toml > $DIR/$MOD/$FNM
# # Creating client's config file
MOD=client
LOC=md
FNM=cfg_${LOC}.json
sed "s/BUCKET/measures_${FLG}_${j}_${i}/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TOKEN/$TKN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DESTINATIONS/upf_${LOC}:4040/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/cli-${LOC}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/1000/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LOGGER/$LGR/g" $DIR/$MOD/$FNM
# Creating client application configuration
FNM=app_${LOC}.toml
sed "s/DESTINATIONS/upf_${LOC}:4040/g" $SRCDIR/$MOD/app.toml > $DIR/$MOD/$FNM
LOC=trn
FNM=cfg_${LOC}.json
sed "s/BUCKET/measures_${FLG}_${j}_${i}/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TOKEN/$TKN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DESTINATIONS/upf_${LOC}:4040/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/cli-${LOC}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/1000/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LOGGER/logger:8086/g" $DIR/$MOD/$FNM
# Creating client application configuration
FNM=app_${LOC}.toml
sed "s/DESTINATIONS/upf_${LOC}:4040/g" $SRCDIR/$MOD/app.toml > $DIR/$MOD/$FNM
# # Creating background's config file
LOC=md
MOD=background
FNM=cfg_${LOC}.json
sed "s/DESTINATIONS/srv_${LOC}:4040/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/${LOC}-bg/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/${RPS}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/toml\",/toml\"/g" $DIR/$MOD/$FNM
# Creating client application configuration
FNM=app_${LOC}.toml
sed "s/DESTINATIONS/srv_${LOC}:4040/g" $SRCDIR/$MOD/app.toml > $DIR/$MOD/$FNM
LOC=trn
FNM=cfg_${LOC}.json
sed "s/DESTINATIONS/srv_${LOC}:4040/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/${LOC}-bg/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/${RPS}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/toml\",/toml\"/g" $DIR/$MOD/$FNM
# Creating client application configuration
FNM=app_${LOC}.toml
sed "s/DESTINATIONS/srv_${LOC}:4040/g" $SRCDIR/$MOD/app.toml > $DIR/$MOD/$FNM
# Creating routing's config file
MOD=hop
LOC=md
FNM=cfg_${LOC}.toml
sed "s/DST/${SRV_MD}:${PT_MD_SRV}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_UP/no delay/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_DN/${LTDNH_MD}/g" $DIR/$MOD/$FNM
LOC=upf_md
FNM=cfg_${LOC}.toml
sed "s/DST/${SRV_MD}:${PT_MD_SRV}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_UP/${LTUP_UPF}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_DN/${LTDN_UPF}/g" $DIR/$MOD/$FNM
LOC=trn
FNM=cfg_${LOC}.toml
sed "s/DST/${SRV_TRN}:${PT_TRN_SRV}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_UP/${LTUPH_TRN}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_DN/${LTDNH_TRN}/g" $DIR/$MOD/$FNM
LOC=upf_trn
FNM=cfg_${LOC}.toml
sed "s/DST/${SRV_TRN}:${PT_TRN_SRV}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_UP/${LTUP_UPF}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LTY_DN/${LTDN_UPF}/g" $DIR/$MOD/$FNM

# Creating .env file
# InfluxDB
sed "s/TKN/$TKN/g" $SRCDIR/env > ./.env
sed "${OPTS[@]}" "s/BKT/measures_${FLG}_${j}_${i}/g" ./.env
LGRHST="$(echo $LGR | sed 's/"//g')"
sed "${OPTS[@]}" "s/LGR/${LGRHST}/g" ./.env
# Client
LOC=md
EPATH=$( echo $DIR/client/app_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_A_MD/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/cfg_${LOC}.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_C_MD/${EPATH}/g" ./.env
LOC=trn
EPATH=$( echo $DIR/client/app_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_A_TRN/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/cfg_${LOC}.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_C_TRN/${EPATH}/g" ./.env

# Background
LOC=md
EPATH=$( echo $DIR/background/app_${LOC}.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/BG_A_MD/${EPATH}/g" ./.env
EPATH=$( echo $DIR/background/cfg_${LOC}.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/BG_C_MD/${EPATH}/g" ./.env
LOC=trn
EPATH=$( echo $DIR/background/app_${LOC}.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/BG_A_TRN/${EPATH}/g" ./.env
EPATH=$( echo $DIR/background/cfg_${LOC}.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/BG_C_TRN/${EPATH}/g" ./.env

# Hop
LOC=md
EPATH=$( echo $DIR/hop/cfg_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/HP_C_MD/${EPATH}/g" ./.env
sed "${OPTS[@]}" "s/HP_D_MD/srv_${LOC}:4040/g" ./.env
sed "${OPTS[@]}" "s/HP_S_MD/${SRV_MD}:${PT_MD_SRV}/g" ./.env
LOC=upf_md
EPATH=$( echo $DIR/hop/cfg_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/UPF_C_MD/${EPATH}/g" ./.env
sed "${OPTS[@]}" "s/UPF_D_MD/${SRV_MD}:${PT_MD_DEL}/g" ./.env
sed "${OPTS[@]}" "s/UPF_S_MD/${SRV_MD}:${PT_MD_SRV}/g" ./.env
LOC=trn
EPATH=$( echo $DIR/hop/cfg_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/HP_C_TRN/${EPATH}/g" ./.env
sed "${OPTS[@]}" "s/HP_D_TRN/srv_${LOC}:4040/g" ./.env
sed "${OPTS[@]}" "s/HP_S_TRN/${SRV_TRN}:${PT_TRN_SRV}/g" ./.env
LOC=upf_trn
EPATH=$( echo $DIR/hop/cfg_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/UPF_C_TRN/${EPATH}/g" ./.env
sed "${OPTS[@]}" "s/UPF_D_TRN/${LOC}:4040/g" ./.env
sed "${OPTS[@]}" "s/UPF_S_TRN/${SRV_TRN}:${PT_TRN_SRV}/g" ./.env

# Server
LOC=trn
EPATH=$( echo $DIR/server/cfg_${LOC}.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/SRV_A_TRN/${EPATH}/g" ./.env
sed "${OPTS[@]}" "s/SRV_D_TRN/${SRV_MD}:${PT_MD_SRV}/g" ./.env
LOC=md
EPATH=$( echo $DIR/server/cfg_${LOC}.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/SRV_A_MD/${EPATH}/g" ./.env
sed "${OPTS[@]}" "s/SRV_D_MD/${SRV_TRN}:${PT_TRN_SRV}/g" ./.env
