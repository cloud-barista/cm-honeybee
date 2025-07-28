#!/bin/bash

### Firewalld Rules
sudo systemctl enable firewalld --now

sudo firewall-cmd --permanent --zone=public --add-port=22/tcp
sudo firewall-cmd --permanent --zone=public --add-port=80/tcp
sudo firewall-cmd --permanent --zone=public --add-port=443/tcp

sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="8086" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="8888" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9201" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9202" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9203" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9204" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9206" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="3100" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="3000" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="8443" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9000" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9001" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="18080" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="13000" protocol="tcp" accept'

sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9101" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9100" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9106" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9105" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9000" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="8080" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9102" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9103" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="9104" protocol="tcp" accept'

sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="5672" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="1883" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="4369" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="15672" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="15675" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="25672" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="8883" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="16567" protocol="tcp" accept'
sudo firewall-cmd --permanent --zone=public --add-rich-rule='rule family="ipv4" source address="192.168.110.0/24" port port="8000" protocol="tcp" accept'

sudo firewall-cmd --reload

### ufw Rules
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

sudo ufw allow from 192.168.110.0/24 to any port 8086 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 8888 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9201 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9202 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9203 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9204 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9206 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 3100 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 3000 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 8443 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9000 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9001 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 18080 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 13000 proto tcp

sudo ufw allow from 192.168.110.0/24 to any port 9101 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9100 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9106 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9105 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 8080 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9102 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9103 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 9104 proto tcp

sudo ufw allow from 192.168.110.0/24 to any port 5672 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 1883 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 4369 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 15672 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 15675 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 25672 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 8883 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 16567 proto tcp
sudo ufw allow from 192.168.110.0/24 to any port 8000 proto tcp

### iptables Rules
sudo iptables -D INPUT -p tcp -m multiport --dports 8081,8082 -j ACCEPT
sudo iptables -A INPUT -p tcp -m multiport --dports 8081,8082 -j ACCEPT
sudo iptables -D INPUT -s 192.168.110.0/24 -p udp --dport 53 -j ACCEPT
sudo iptables -A INPUT -s 192.168.110.0/24 -p udp --dport 53 -j ACCEPT
