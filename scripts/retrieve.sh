#!/usr/bin/env bash
set -xe

usage="$(basename "$0") [-h] [-i PROJECT] [-v VM] [-p PYTHON] [-d NOTEBOOKS]
Retrieve data from DB for a (group of) experiment(s):
    -f  Idetifier of the scenario
    -t  Traffic uplink/downlink or in/out
    -s  Solution used (game/opt)
    -T  InfluxDB token
    -k  Key used to query the DB
    -n  Min index for the experiment
    -x  Max index for the experiment
    -N  Min index for the scenario within an experiment
    -X  Max index for the scenario within an experiment
    -v  Path where the host voulme is attached
    -d  Output directory"

while getopts "f:t:s:T:d:e:n:x:N:X:v:o:" opt; do
	case $opt in
		f) FLG=$OPTARG;;    # Idetifier of the scenario
		t) TYPE=$OPTARG;;   # Traffic uplink/downlink or in/out
		s) SOL=$OPTARG;;    # Solution used (game/opt)
		T) TKN=$OPTARG;;    # InfluxDB token
		d) DEV=$OPTARG;;    # idDevice used to query the DB
		e) EVT=$OPTARG;;    # eventType used to query the DB
		n) MIN_R=$OPTARG;;  # Min index for the experiment
		x) MAX_R=$OPTARG;;  # Max index for the experiment
		N) MIN_S=$OPTARG;;  # Min index for the scenario within an experiment
		X) MAX_S=$OPTARG;;  # Max index for the scenario within an experiment
		v) VOL=$OPTARG;;    # Path where the host voulme is attached
		o) OUT=$OPTARG;;    # Output directory
		*)echo 'Error in command line parsing' >&2;
        exit 1;;
	esac
done

if [ ! "$FLG" ] || [ ! "$TYPE" ] || [ ! "$SOL" ] || [ ! "$DEV" ] || [ ! "$EVT" ] || [ ! "$MIN_R" ] || [ ! "$MAX_R" ] || [ ! "$MIN_S" ] || [ ! "$MAX_S" ]; then
  echo "arguments -f, -t, -s, -d, -e, -n, -x, -N, -X must be provided"
  echo "$usage" >&2; exit 1
fi

BKT="measures_${FLG}_${SOL}"

if [ "$TKN" ]; then
    influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito
fi

for j in $(seq $MIN_R $MAX_R); do
    if [ "$VOL" ]; then
        BASE=${VOL}
    fi
    if [ "$OUT" ]; then
        BASE+=/$OUT
    fi
    DIR=${BASE}/${FLG}/${SOL}/${j}
    mkdir -p $DIR
    for i in $(seq $MIN_S $MAX_S);do 
	    echo "extract ${DEV} data from bucket ${BKT}_${j}_${i}"
	    influx query --raw "from(bucket:\"${BKT}_${j}_${i}\") |> range(start: -365d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |> filter(fn: (r) => r[\"idDevice\"] == \"${DEV}\") |> filter(fn: (r) => r[\"eventType\"] == \"$EVT\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > ${FLG}_${TYPE}_${SOL}_${DEV}_${j}_${i}.csv;
        mv ${FLG}_${TYPE}_${SOL}_${DEV}_${j}_${i}.csv $DIR
        echo "${DIR}/${FLG}_${TYPE}_${SOL}_${DEV}_${j}_${i}.csv"
    done;
done;
