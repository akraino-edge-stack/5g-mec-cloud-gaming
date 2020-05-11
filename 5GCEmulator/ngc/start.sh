#!/bin/bash
####################### Block diagram of the Emuloator #######################################
#           Indicator file (on) <-- NEF <--> AF <--> Curl (Post/Delete)
#                    ^
#                    |
#                    |
#                    | 
#                   UPF (iperf client)   <----->  MEC server (iperf svr)
##############################################################################################
###Offloading phase -- run ./test/Post.sh to create the Indicator file("on"). Subsequently, iperf is periodically executed to emulate a client accessing the MEC server.
###Non-offloading phase -- run ./test/Delete.sh to clear the Indicator file ("off") to stop offloading.
###
Emulator_path="/tmp/5GCEmulator"
certs_path="/etc/certs"
bin_dir=$PWD
#IP addr of a remote MEC server
svr_addr=127.0.0.1
time=4
UPF_exe="iperf -t $time -u -c $svr_addr"
Svr_exe="iperf -u -s -i 1"
AF_exe="af"
NEF_exe="nef"
state=false
export Emulator_path=$Emulator_path
mkdir -p $Emulator_path
rm -rf $Emulator_path/*
Usage="\
start.sh [-h] [-s srv_ip]\n
   -h print the help message.\n
   -s specify the ip of a remote MEC server.\n
   In case of no options, an iperf instance will be launched in the server mode to emulate a local MEC server.
"
#Process cmdline arguments
islocal=true
for opt
do
	case $opt in
		-h)
			echo -e $Usage; exit 0;;
		-s)
			srv_addr=$2; islocal=false;;
		-*)
			echo "Illegal options"; exit 1;;
	esac
done
#Check and install certs
mkdir -p "$certs_path"
if [ -e $certs_path/root-ca-cert.pem ] && [ -e $certs_path/server-cert.pem ] && [ -e $certs_path/server-key.pem ]; then
	echo "Certificates exist"
else
	echo "Installing certificates..."
	sleep $time
	./scripts/genCerts.sh -t DNS -n localhost -m localhost
	rm extfile.cnf root-ca-key.pem root-ca-cert.srl server-request.csr
	mv root-ca-cert.pem server-cert.pem server-key.pem "$certs_path" || { echo "Fail to generate certificates... Please do it manually and re-run start.sh." ; exit 1; }
	sleep $time
	echo "Certificates are installed"
	sleep $time
fi

#Start an iperf instance when srv_ip is not given
if $islocal; then
	echo "Lauching a local server"
	$Svr_exe &
	sleep $time
fi
#start AF and NEF
echo "Launching AF ..."
$bin_dir/$AF_exe &
sleep $time
echo "Launching NEF ..."
$bin_dir/$NEF_exe &
sleep $time
#Start the loop of UPF output
while true; do
	case $state in
		false)
			#Check if the Indicator file exists
                        if [ -e $Emulator_path/on ]; then
                                state=true
                        else
                                echo "No data offloaded..."
                        fi;;
                true)
                        if [ -e $Emulator_path/on ]; then
				if ! $islocal; then
					echo "Offloading in progress..."
				fi
                        $UPF_exe >/dev/null 2>&1
                        #$UPF_exe
                        else state=false
                        fi;;
	esac
        sleep $time
done
