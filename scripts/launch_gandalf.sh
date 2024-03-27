#!/usr/bin/env bash
#
set -xe

# ./scripts/setup_gandalf.sh poa_remote 2 '"130.192.212.176:8085"' XJGkvOmlXk98pofGZW6Krnt05-eV9A669E-gcbPJnOgSmd8H8G4MpYZ9_iMB4aR7Y-zC5ysZrbvh99DE1RIr1A== client 60 '"proxy:4040"' 60 '"proxy:4040"' '"130.192.212.176:4343", "193.147.104.34:43001", "130.192.212.176:4444"' "193.147.34:43002"
flg_exp=${1:?identifier of the experiment}
appto=${2:?application timeout}
lambda_cli=${3:? traffic generated at the clients}
time=${4:? Duration of the experiment}
a1=${5:? Alpha player 1}
a2=${6:? Alpha player 2} 
srv_tme_ty=${7:? Service type (exp|det)}
d=${8:? Games\' round duration}
b=${9:? Bin size}
sync=${10:?synchronize servers time? (yes/no)}

DHOST_TRN="130.192.212.176"
DHOST_MAD="193.147.104.34"

PATH_TRN="/home/arch/git/RoPE"
PATH_MAD="/home/vincenzo/RoPE"

# command parameters
logger=$DHOST_TRN:8086
token=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==
pl_1_id='player_1'
pl_2_id='player_2'
srv_trn=$DHOST_TRN:4343
srv_md=$DHOST_MAD:8083

if [ $sync = "yes" ]; then
	sudo ntpdate 0.europe.pool.ntp.org
	ssh -p 2280 -t vincenzo@$DHOST_MAD sudo -S ntpdate 0.europe.pool.ntp.org
	ssh -t arch@$DHOST_TRN sudo ntpdate 0.europe.pool.ntp.org
fi

./scripts/setup_gandalf.sh -f $flg_exp -l $logger -t $token -d $d -x $appto -I $pl_1_id -A $a1 -i $pl_2_id -a $a2 -r $lambda_cli -T $time -s $srv_trn -s $srv_md -b $b -S $srv_tme_ty


ssh -t arch@$DHOST_TRN "cd ${PATH_TRN}; ./scripts/setup_gandalf.sh -f $flg_exp -l $logger -t $token -d $d -I $pl_1_id -A $a1 -i $pl_2_id -a $a2 -r $lambda_cli -T $time -s $srv_trn -s $srv_md -b $b -S $srv_tme_ty"
ssh -p 2280 -t vincenzo@$DHOST_MAD "cd ${PATH_MAD}; ./scripts/setup_gandalf.sh -f $flg_exp -l $logger -t $token -d $d -I $pl_1_id -A $a1 -i $pl_2_id -a $a2 -r $lambda_cli -T $time -s $srv_trn -s $srv_md -b $b -S $srv_tme_ty"
# Start logger
ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d logger"
# sleep 10
ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d srv_trn"
ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose up -d srv_md"

docker compose up player_1 player_2 &

docker ps -a | awk '{if($2 == "paolocastagno/rope-client:latest") system("docker logs " $1)}'  >> player_logs

sleep $time
docker-compose down
ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose down"
ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose down"
