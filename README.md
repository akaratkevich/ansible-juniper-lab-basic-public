# ansible-juniper-lab-01-public

*Ansible Lab: containing 3x VMX, 1x Linux Routing Daemon (exaBGP)

Lab purpose:
- Explore Ansible automation on Junos OS (*basics)
	- Information Gathering
	- Templating Configuration
	- Deploying Configuration

The idea behind this lab is to explore the basic building blocks behind automating Junos devices using Ansible. 
*Please note, not all elements or protocols are covered in this lab. The lab will be expanded in later revisions.

#### Prerequisites:

**Containerlab (with respective NOS images) and Ansible installed on the control node.
	[Containerlab](https://containerlab.dev/install/)
	[Ansible](https://www.ansible.com/)
		
Please review the installation documentation for the above to ensure all required dependencies are also installed. 

## Breaking Down Lab Components

Network will be created as a multi-node container-based environments using Containerlab.

- ISP
- CORE
- DISTRIBUTION
- *ACCESS (Omitted in this example)
- *ROUTE REFLECTOR (Omitted in this example)
- *CPE (Omitted in this example)

ISP - Consists of one virtual server running exaBGP (* with modified [route-smash script](https://github.com/dfex/route-smash/blob/master/route-smash.py)  to only announce 172.16.x.0/24 ranges)

CORE - Consists of two virtual routers running Juniper vr-vmx, which will be the backbone of the network.

DISTRIBUTION - Consists of one virtual routers running Juniper vr-vmx,

*ACCESS - Consists of one virtual router running Arista cEOS. (Omitted in this example)

*ROUTE REFLECTOR - Consists of one virtual router running Juniper vr-vmx. (Omitted in this example)

*CPE - Consists of one virtual server running Linux/Alpine (Omitted in this example)

#### Connectivity Overview:

ISP - Exterior Gateway Protocol (eBGP) will be used between the external ISP (`isp1`) and the core routers (`core1/core2`). This simulates the peering relationship between the network and an external Internet Service Provider (ISP).

CORE - Exterior Gateway Protocol (eBGP) will be used between  the core routers (`core1/core2`) and external ISP (`isp1`). Interior Gateway Protocol (iBGP) will be used between the core routers (`core1` and `core2`) , the distribution  (`dst1`). 

DISTRIBUTION - ISIS will be used as IGP to propagate loopback ip addresses of core/distribution layers.

*ROUTE REFLECTOR - Will reflects BGP routes received from the core routers (`core1` and `core2`) to the distribution router (`dst1`) and vice versa. The router acts as a central point for iBGP peering, simplifying the iBGP peering relationships within the core layer.

*ACCESS - Layer 2 (L2) Switching: Arista cEOS will handle Layer 2 functionality in the access layer, including Ethernet switching and VLAN handling

## Containerlab Topology

The topology file serves as a blueprint for defining the lab environment, including the devices to be deployed and their connections.

Topology definition file:

<mark style="background: #ADCCFFA6;">topololy.yml</mark>
```
name: automatedNetwork/lab-01
prefix: ""
# containerlab topology for the lab

mgmt:
  network: lab_1
  ipv4-subnet: 172.100.1.0/24

topology:
  kinds:
    vr-vmx:
      image: vrnetlab/vr-vmx:22.3R1.11
    linux:
      image: isp-exabgp-172:latest

  nodes:
    isp1:
      kind: linux
      mgmt-ipv4: 172.100.1.2

    core1:
      kind: vr-vmx
      mgmt-ipv4: 172.100.1.3

    core2:
      kind: vr-vmx
      mgmt-ipv4: 172.100.1.4
    
    dst1:
      kind: vr-vmx
      mgmt-ipv4: 172.100.1.5
      

  # Define links (interconnections)
  links:
    - endpoints: ["isp1:eth1", "core1:eth1"]
    - endpoints: ["isp1:eth2", "core2:eth1"]
    - endpoints: ["core1:eth2", "core2:eth2"]
    - endpoints: ["core1:eth3", "dst1:eth1"]
    - endpoints: ["core2:eth3", "dst1:eth2"]
```

Following docker images are used in the lab:
- <mark style="background: #CACFD9A6;">vrnetlab/vr-vmx:22.3R1.11</mark> [vrnetlab](https://github.com/hellt/vrnetlab.git)
- <mark style="background: #CACFD9A6;"> isp-exabgp-172:latest </mark> Modified docker image based on [exabgp](https://github.com/Exa-Networks/exabgp)

#### Docker Image exaBGP

I have modified the exaBGP configuration file to suit my topology in this lab example as well as the python script that announces the routes and put it all together into a docker image as follows;

A Dockerfile is a text file that contains instructions for building a Docker image:

<mark style="background: #ADCCFFA6;">Dockerfile</mark>
```
FROM ubuntu:22.04

# Install dependencies
COPY exa.cfg .
COPY route-smash-172.py .
RUN apt update
RUN apt install python3-pip net-tools wget mrtparse vim nano -y && \
    rm -rf /var/lib/apt/lists/* && apt clean
RUN pip install exabgp==4.2.17
```

This is the configuration file for exaBGP:
<mark style="background: #ADCCFFA6;">exa.cfg</mark>
```
process announce-routes {
    run python3 ./route-smash-172.py;
    encoder json;
}

neighbor 10.0.1.1 {                 # Remote neighbor to peer with
    router-id 10.10.10.10;          # Local router-id
    local-address 10.0.1.10;        # Local update-source
    local-as 65000;                 # Local AS
    peer-as 65001;                  # Peer's AS

    api {
        processes [announce-routes];
    }
}
neighbor 10.0.2.1 {                 # Another remote neighbor to peer with
    router-id 10.10.10.10;          # Local router-id for the new neighbor
    local-address 10.0.2.10;        # Local update-source for the new neighbor
    local-as 65000;                 # Local AS for the new neighbor
    peer-as 65001;                  # Peer's AS for the new neighbor

    api {
        processes [announce-routes];
    }
}
```

This is the python script that announces the routes using exaBGP:
<mark style="background: #ADCCFFA6;">route-smash-172.py</mark>
```
#!/usr/bin/env python

## routesmash.py - 29/07/2014
## ben.dale@gmail.com
## Spam a list of generated /24 prefixes
## Use with ExaBGP for load testing

import sys
import time

for third in range(0, 255):
    sys.stdout.write('announce route 172.16.%d.0/24 next-hop 10.10.10.10\n' % third)
    sys.stdout.flush()
    ## Back off timer if router is too slow:
    ## time.sleep(0.001)

while True:
    time.sleep(1)
```
*This has been modified from the original [route-smash](https://github.com/dfex/route-smash/blob/master/route-smash.py)  to only announce 172.16.x.0/24

Once these three files are in the same directory, build the Docker image by running the following command:

```
docker build -t isp-exabgp-172 .
```

*At this stage you can spin-up the lab with <mark style="background: #FF5582A6;">no configuration</mark> using Containerlab.

To deploy the lab, from the same directory where the clab topology.yml is located issue the following command:
```
sudo clab deploy -t topology.yml 
```


```
anton@mcc:~/automatedNetwork/lab-01/clab$ sudo clab deploy -t topology.yml 
INFO[0000] Containerlab v0.42.0 started                 
INFO[0000] Parsing & checking topology file: topology.yml 
INFO[0000] Creating lab directory: /home/anton/automatedNetwork/lab-01/clab/clab-automatedNetwork/lab-01 
INFO[0000] Creating docker network: Name="lab_1", IPv4Subnet="172.100.1.0/24", IPv6Subnet="", MTU="1500" 
INFO[0000] Creating container: "isp1"                   
INFO[0000] Creating container: "core2"                  
INFO[0000] Creating container: "dst1"                   
INFO[0000] Creating container: "core1"                  
INFO[0000] Creating virtual wire: core1:eth2 <--> core2:eth2 
INFO[0001] Creating virtual wire: core2:eth3 <--> dst1:eth2 
INFO[0001] Creating virtual wire: core1:eth3 <--> dst1:eth1 
INFO[0001] Creating virtual wire: isp1:eth2 <--> core2:eth1 
INFO[0001] Creating virtual wire: isp1:eth1 <--> core1:eth1 
INFO[0001] Adding containerlab host entries to /etc/hosts file 
+---+-------+--------------+---------------------------+--------+---------+----------------+--------------+
| # | Name  | Container ID |           Image           |  Kind  |  State  |  IPv4 Address  | IPv6 Address |
+---+-------+--------------+---------------------------+--------+---------+----------------+--------------+
| 1 | core1 | cd2e04e88e42 | vrnetlab/vr-vmx:22.3R1.11 | vr-vmx | running | 172.100.1.3/24 | N/A          |
| 2 | core2 | 2111d1883dcd | vrnetlab/vr-vmx:22.3R1.11 | vr-vmx | running | 172.100.1.4/24 | N/A          |
| 3 | dst1  | 7314dcaab8bd | vrnetlab/vr-vmx:22.3R1.11 | vr-vmx | running | 172.100.1.5/24 | N/A          |
| 4 | isp1  | ab6edf259fbe | isp-exabgp-172:latest     | linux  | running | 172.100.1.2/24 | N/A          |
+---+-------+--------------+---------------------------+--------+---------+----------------+--------------+
```

Containerlab adds the hosts entries to /etc/hosts file so you can use hostname to connect to the nodes.

To connect to vr-vmx:
```
ssh admin@[hostname/ip]
```
Default password is: <mark style="background: #FFB86CA6;">admin@123</mark>

To connect to exabgp:
```
docker exec -it isp1 bash
```

Once connected to the `isp1` please remember to configure and enable interfaces and start the daemon

```
ifconfig eth1 10.0.1.10 netmask 255.255.255.0 up
ifconfig eth2 10.0.2.10 netmask 255.255.255.0 up
exabgp ./exa.cfg
```


## Ansible

The next section explores automation of configuration deployment and network reconnaissance.

I have structured this lab directory as follows:

```
anton@mcc:~/automatedNetwork/lab-01$ tree -L 3
.
├── ansible.cfg
├── clab
│   ├── clab-automatedNetwork
│   ├── topology1.yml
│   └── topology.yml
├── configuration
│   ├── core1-eBGP.txt
│   ├── core1.txt
│   ├── core2-eBGP.txt
│   ├── core2.txt
│   ├── dst1.txt
│   └── sshRSA.cfg
├── docker
│   ├── Dockerfile
│   ├── exa.cfg
│   ├── notes-route-smash
│   ├── route-smash-172.py
│   └── route-smash.py
├── docs
├── inventory
│   └── inventory.yml
├── playbooks
│   ├── conf-eBGP-file.yml
│   ├── conf-file.yml
│   ├── conf-line.yml
│   ├── conf-netconf.yml
│   ├── conf-rollback-1.yml
│   ├── facts-config-json.yml
│   ├── facts-config.yml
│   ├── facts-hostname.yml
│   ├── facts-version.yml
│   ├── group_vars
│   │   └── vr-vmx.yml
│   ├── host_vars
│   │   ├── core1.yml
│   │   ├── core2.yml
│   │   └── dst1.yml
│   ├── per-node-eBGP-tasks.yml
│   ├── render-conf-eBGP.yml
│   └── render-eBGP-template.yml
├── scripts
│   └── deploy-conf-file
├── tasks
└── templates
    └── eBGP.j2

12 directories, 33 files
```

### Ansible Configuration

At the root of the directory is my Ansible configuration file:

<mark style="background: #ADCCFFA6;">ansible.cfg</mark>
```
[defaults]
inventory = inventory
host_key_checking = False
```

*More information on building the [configuration file](https://docs.ansible.com/ansible/latest/reference_appendices/config.html)

At this stage the above settings are sufficient for the purpose of this lab. Other folders/files will be explored updated through out the lab.

### Ansible Inventory

The next stage is to put together an inventory file for Ansible. Again there are various ways to achieve that, but in this example I have placed the inventory file into the lab directory which can be passed to Ansible using a <mark style="background: #FFB86CA6;">-i </mark><filename.yml> flag. This allows me to have different inventory files for different labs.

```
anton@mcc:~/automatedNetwork/lab-01$ tree ./inventory/
./inventory/
└── inventory.yml

0 directories, 1 file
```

<mark style="background: #ADCCFFA6;">inventory.yml </mark>
```
LAB:
  children:
    INTERNET:
      hosts:
        isp1:
          ansible_host: 172.100.1.2
    CORE:
      hosts:
        core1:
          ansible_host: 172.100.1.3
        core2:
          ansible_host: 172.100.1.4
    DST:
      hosts:
        dst1:
          ansible_host: 172.100.1.5

vr-vmx:
  children:
    CORE:
    DST:
```

To view the hierarchical grouping of the inventory in Ansible, run the following command:
```

anton@mcc:~/automatedNetwork/lab-01$ ansible-inventory -i inventory/inventory.yml --graph
@all:
  |--@ungrouped:
  |--@LAB:
  |  |--@INTERNET:
  |  |  |--isp1
  |  |--@CORE:
  |  |  |--core1
  |  |  |--core2
  |  |--@DST:
  |  |  |--dst1
  |--@vr-vmx:
  |  |--@CORE:
  |  |  |--core1
  |  |  |--core2
  |  |--@DST:
  |  |  |--dst1
```

### SSH RSA

In Ansible, SSH RSA key pairs are commonly used for authentication when connecting to remote hosts. SSH key-based authentication is a secure and convenient method for establishing a connection between the Ansible control node and the managed hosts

The sample of the required Juniper configuration to be deployed is: 
<mark style="background: #ADCCFFA6;">sshRSA.cfg</mark>
```
configure
set system login user anton uid 2001
set system login user anton class super-user
set system login user anton authentication ssh-rsa "ssh-rsa ###KEY### anton@mcc"
commit-and-quit
```

I have put together a basic Go script to push this configuration to the nodes rather than manually uploading them to each node..
```
package main

import (
 "bufio"
 "flag"
 "fmt"
 "io/ioutil"
 "log"
 "os"
 "strings"

 "golang.org/x/crypto/ssh"
)

func main() {
 // SSH connection details
 username := "admin"
 password := "admin@123"

 // Configuration file path (input from the user as a command-line argument)
 configFilePath := ""
 flag.StringVar(&configFilePath, "config", "", "Path to the configuration file")
 flag.Parse()

 // Check if the configuration file path is provided
 if configFilePath == "" {
  fmt.Println("Error: Configuration file path is required. Please use the '-config' flag.")
  os.Exit(1)
 }

 // Read the configuration file
 configData, err := os.ReadFile(configFilePath)
 if err != nil {
  log.Fatalf("Failed to read configuration file: %s", err)
 }

 // Prompt the user to enter the Juniper nodes (comma-separated)
 fmt.Print("Enter the Juniper nodes (IP addresses or hostnames):seperated by comma\n")
 reader := bufio.NewReader(os.Stdin)
 nodesInput, _ := reader.ReadString('\n')
 nodesInput = strings.TrimSuffix(nodesInput, "\n")
 nodes := strings.Split(nodesInput, ",")

 // Iterate over each Juniper node
 for _, node := range nodes {
  node = strings.TrimSpace(node)

  // Create the SSH configuration
  sshConfig := &ssh.ClientConfig{
   User: username,
   Auth: []ssh.AuthMethod{
    ssh.Password(password),
   },
   HostKeyCallback: ssh.InsecureIgnoreHostKey(),
  }

  // Connect to the Juniper node
  client, err := ssh.Dial("tcp", node+":22", sshConfig)
  if err != nil {
   log.Printf("Failed to connect to %s: %s", node, err)
   continue
  }

  // Create a new session
  session, err := client.NewSession()
  if err != nil {
   log.Printf("Failed to create SSH session to %s: %s", node, err)
   client.Close()
   continue
  }

  // Get the session's standard input
  stdin, err := session.StdinPipe()
  if err != nil {
   log.Printf("Failed to get session's standard input: %s", err)
   session.Close()
   client.Close()
   continue
  }

  // Start the remote shell
  err = session.Shell()
  if err != nil {
   log.Printf("Failed to start shell on %s: %s", node, err)
   stdin.Close()
   session.Close()
   client.Close()
   continue
  }

  // Write the configuration data to the session's standard input
  go func() {
   defer stdin.Close()
   fmt.Fprintln(stdin, string(configData))
   fmt.Fprintln(stdin, "commit")
  }()

  // Wait for the session to finish
  err = session.Wait()
  if err != nil {
   log.Printf("Failed to execute configuration on %s: %s", node, err)
   continue
  }

  fmt.Printf("Successfully executed configuration on %s\n", node)

  // Close the session and client
  session.Close()
  client.Close()
 }
}
```

And then compiled the script and moved it to <mark style="background: #CACFD9A6;">~/automatedNetwork/lab-01/scripts </mark>and have the SSH RSA config file in <mark style="background: #CACFD9A6;">~/automatedNetwork/lab-01/configuration/</mark>

It can now be executed as follows with the <mark style="background: #FFB86CA6;">-config </mark>flag to pass the ssh/rsa config file:

```
anton@mcc:~/automatedNetwork/lab-01/scripts$ ./deploy-conf-file -config ~/automatedNetwork/lab-01/configuration/sshRSA.cfg 
Enter the Juniper nodes (IP addresses or hostnames):seperated by comma
core1,core2,dst1
Successfully executed configuration on core1
Successfully executed configuration on core2
Successfully executed configuration on dst1
anton@mcc:~/automatedNetwork/lab-01/scripts$
```

This enables us to run Ansible playbooks against the nodes using SSH RSA keys for authentication.

### Information Gathering

To start writing and using playbooks we need to download Ansible modules that we can use to perform operational and configuration tasks on the devices.  Juniper modules are distributed through number of collections and roles, for example: 

To install <mark style="background: #CACFD9A6;">juniper.device </mark>collection from the Ansible Galaxy website, issue the ansible-galaxy collection install command and specify the <mark style="background: #CACFD9A6;">juniper.device</mark> collection:

```
ansible-galaxy collection install juniper.device
```

To install the <mark style="background: #CACFD9A6;">Juniper.junos</mark> role from the Ansible Galaxy website, issue the ansible-galaxy install command and specify the <mark style="background: #CACFD9A6;">Juniper.junos</mark> role.

*Please note: Ansible galaxy is upgrading to collections and plans to deprecate roles in future
```
ansible-galaxy install Juniper.junos
```


To view the installed collections and roles use the following:
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-galaxy collection list | grep juniper
juniper.device                1.0.2  
junipernetworks.junos         4.1.0  
junipernetworks.junos         4.1.0  
junipernetworks.junos         5.2.0 

anton@mcc:~/automatedNetwork/lab-01$ ansible-galaxy role list | grep Juniper
- Juniper.junos, 2.4.3
```

You can view all available Ansible Galaxy collections and roles for Junos [here](https://galaxy.ansible.com/search?deprecated=false&keywords=junos&order_by=-relevance&page=1)

###  [juniper_junos_facts Module](https://junos-ansible-modules.readthedocs.io/en/2.4.0/) 

Before running the first playbook, I have declared some <mark style="background: #CACFD9A6;">group-vars </mark>that will apply to all the nodes in the <mark style="background: #CACFD9A6;">vr-vmx</mark> group (ie Junos nodes) in the inventory file. This is the current directory structure for playbooks folder where I pass those variables to the playbooks:

```
anton@mcc:~/automatedNetwork/lab-01$ tree ./playbooks/ -L 2
./playbooks/
├── enableNetconf.yml
├── facts-hostname.yml
├── group_vars
│   └── vr-vmx.yml
└── host_vars
    ├── core1.yml
    ├── core2.yml
    └── dst1.yml
```

The variables are passed to <mark style="background: #CACFD9A6;">vr-vmx </mark>group in the following file:
<mark style="background: #ADCCFFA6;">vr-vmx.yml </mark>
```
---
ansible_user: anton
ansible_ssh_private_key_file: /home/anton/.ssh/id_rsa
ansible_network_os: junipernetworks.junos.junos
```

This is just the authentication and connection parameters, so that I don't have to declare them in the playbooks.

I think any sensible automation journey should begin with information gathering. Hence its important to put together number of tactical playbooks that will provide you with the necessary information about the network. 

Lets explore <mark style="background: #CACFD9A6;">juniper_junos_facts </mark>module. Although this playbook is pretty moot, as we already know all the hostnames, nonetheless it explores the module and how to access specific keys in that module's dictionary.

Facts collected from the Junos device are from dictionary that contains the keys listed PyEZ's fact gathering system. See [PyEZ facts](http://junos-pyez.readthedocs.io/en/stable/jnpr.junos.facts.html) for a complete list of these keys and their meaning.

This playbook by default will be able to run against all the hosts in the <mark style="background: #CACFD9A6;">vr-vmx </mark>group.

The Juniper Networks modules do not require Python on devices running Junos OS because they use Junos PyEZ and the Junos XML API over NETCONF to interface with the device. Therefore, to perform actions on devices running Junos OS, you must run modules locally on the Ansible control node, where Python is installed. You can run the modules locally by including <mark style="background: #CACFD9A6;">connection: local</mark>  in the playbook.

The <mark style="background: #CACFD9A6;">juniper.device</mark> collection modules also support <mark style="background: #CACFD9A6;">connection: juniper.device.pyez </mark>for establishing a persistent connection to a host to maintain the connection while executing multiple tasks.

<mark style="background: #ADCCFFA6;">facts-hostname.yml</mark>
```
---
- name: Get specific Junos facts
  hosts: vr-vmx
  connection: local
  gather_facts: no
  roles:
    - Juniper.junos
  tasks:
    - name: Get Junos facts
      juniper_junos_facts:
      register: junos_facts

    - name: Print specific fact (Hostname)
      debug:
        var: junos_facts.ansible_facts.junos.hostname
```

You can dry run the play by using <mark style="background: #FFB86CA6;">--check</mark> flag.
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-playbook ./playbooks/facts-hostname.yml -i ./inventory/inventory.yml --check

PLAY [Get specific Junos facts] ******************************************************************************************************************************************

TASK [Get Junos OS version] **********************************************************************************************************************************************
ok: [core1]
ok: [dst1]
ok: [core2]

TASK [Print facts] *******************************************************************************************************************************************************
ok: [core1] => {
    "junos_facts.ansible_facts.junos.hostname": "core1"
}
ok: [dst1] => {
    "junos_facts.ansible_facts.junos.hostname": "dst1"
}
ok: [core2] => {
    "junos_facts.ansible_facts.junos.hostname": "core2"
}

PLAY RECAP ***************************************************************************************************************************************************************
core1                      : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
core2                      : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
dst1                       : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0 
```

Lets run the playbook but limit the execution to CORE nodes only using <mark style="background: #FFB86CA6;">--limit</mark> flag:
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-playbook ./playbooks/facts-hostname.yml -i ./inventory/inventory.yml --limit=CORE

PLAY [Get specific Junos facts] ******************************************************************************************************************************************

TASK [Get Junos OS version] **********************************************************************************************************************************************
ok: [core1]
ok: [core2]

TASK [Print facts] *******************************************************************************************************************************************************
ok: [core1] => {
    "junos_facts.ansible_facts.junos.hostname": "core1"
}
ok: [core2] => {
    "junos_facts.ansible_facts.junos.hostname": "core2"
}

PLAY RECAP ***************************************************************************************************************************************************************
core1                      : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
core2                      : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

Organising your inventory, allows you to granularly limit the execution of a play down to a specific host.  

Lets look at few more modules useful for information gathering.

### [juniper.device Collection](https://galaxy.ansible.com/juniper/device)  

This example uses <mark style="background: #CACFD9A6;">command</mark> module from <mark style="background: #CACFD9A6;">juniper.device</mark> collection  This module can execute one or more CLI commands on a Junos device. 

<mark style="background: #ADCCFFA6;">facts-config.yml</mark>
```
- name: Get specific Junos facts / Config
  hosts: vr-vmx
  connection: local
  gather_facts: no
  collections:
    - juniper.device
  
  tasks:
    - name: Get Junos OS version
      command:
        commands: "show configuration | display set"
      register: junos_facts

    - name: Print facts
      debug:
        var: junos_facts
```

###  [junipernetworks.junos Collection](https://galaxy.ansible.com/junipernetworks/junos) 

This example uses <mark style="background: #CACFD9A6;">junos_facts </mark>module to collect facts from remote devices running Junos

<mark style="background: #ADCCFFA6;">facts-config-json.yml</mark>
```
---
- name: collect configuration
  hosts: vr-vmx
  connection: local
  gather_facts: no
  vars:
    ansible_connection: ansible.netcommon.netconf
  tasks:
    - name: Get configuration
      junipernetworks.junos.junos_facts:
        gather_subset: config
        config_format: json
      register: output

    - name: Display config
      debug:
        var: output
```

Now that we have the basics for information gathering, we can move on to configuring devices using Ansible.

*More reconnaissance playbooks can be put together later based on the specific needs using similar modules and methods.

### Configuration Management

Lets explore some basics around deploying configuration to the Junos nodes using Ansible.

### [Juniper.junos Role](https://junos-ansible-modules.readthedocs.io/en/2.4.0/)

This example uses <mark style="background: #CACFD9A6;">juniper_junos_config</mark> module:

#### Load from file

I have the following 3x files prepped in the configuration directory
```
anton@mcc:~/automatedNetwork/lab-01/configuration$ tree
.
├── core1.txt
├── core2.txt
├── dst1.txt
```


Example <mark style="background: #ADCCFFA6;">core1.txt</mark>
```
set interfaces ge-0/0/0 description "## Link to ISP1 (eth1)## "
set interfaces ge-0/0/0 unit 0 family inet address 10.0.1.1/24
set interfaces ge-0/0/1 description "## Link to core2 (eth2)## "
set interfaces ge-0/0/1 unit 0 family inet address 10.0.3.1/24
set interfaces ge-0/0/2 description "## Link to dst1 (eth3)## "
set interfaces ge-0/0/2 unit 0 family inet address 10.0.4.1/24
set interfaces lo0 description "## System_Loopback ##"
set interfaces lo0 unit 0 family inet address 1.1.1.1/32
set protocols lldp management-address 1.1.1.1
set protocols lldp ptopo-configuration-trap-interval 60
set protocols lldp lldp-configuration-notification-interval 60
set protocols lldp port-id-subtype interface-name
set protocols lldp interface all disable
set protocols lldp interface ge-0/0/1
set protocols lldp interface ge-0/0/2
```
*We are not templating the configuration yet :)

<mark style="background: #ADCCFFA6;">conf-file.yml </mark>
```
---
- name: Load and commit configuration file
  hosts: vr-vmx
  gather_facts: false
  connection: local
  roles:
    - Juniper.junos
  tasks:
    - name: Load configuration from a local file and commit
      juniper_junos_config:
        load: merge
        format: set
        src: "./configuration/{{ inventory_hostname.split('.')[0] }}.txt"
      register: response

    - name: Print the response
      debug:
        var: response
```

To run this playbook:
```
ansible-playbook ./playbooks/conf-file.yml -i ./inventory/inventory.yml
```

Result: 
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-playbook ./playbooks/conf-file.yml -i ./inventory/inventory.yml 

PLAY [Load and commit configuration file] ************************************************************************************************

TASK [Load configuration from a local file and commit] ***********************************************************************************
changed: [core2]
changed: [core1]
changed: [dst1]

TASK [Print the response] ****************************************************************************************

---->>>> OMIITED


PLAY RECAP *******************************************************************************************************************************
core1                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
core2                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
dst1                       : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0  

```

Lets look on one of the nodes:

The configuration file has been successfully pushed out to the nodes.
```
anton@core1> show interfaces descriptions    
Interface       Admin Link Description
ge-0/0/0        up    up   ## Link to ISP1 (eth1)## 
ge-0/0/1        up    up   ## Link to core2 (eth2)## 
ge-0/0/2        up    up   ## Link to dst1 (eth3)## 
lo0             up    up   ## System_Loopback ##

anton@core1> show lldp neighbors             
Local Interface    Parent Interface    Chassis Id          Port info          System Name
ge-0/0/1           -                   2c:6b:f5:4b:61:c0   ge-0/0/1           core2               
ge-0/0/2           -                   2c:6b:f5:cf:ab:c0   ge-0/0/0           dst1                

anton@core1> ping 10.0.1.1 
PING 10.0.1.1 (10.0.1.1): 56 data bytes
64 bytes from 10.0.1.1: icmp_seq=0 ttl=64 time=0.405 ms
64 bytes from 10.0.1.1: icmp_seq=1 ttl=64 time=0.033 ms
^C
--- 10.0.1.1 ping statistics ---
2 packets transmitted, 2 packets received, 0% packet loss
round-trip min/avg/max/stddev = 0.033/0.219/0.405/0.186 ms

anton@core1> ping 10.0.3.2    
PING 10.0.3.2 (10.0.3.2): 56 data bytes
64 bytes from 10.0.3.2: icmp_seq=0 ttl=64 time=23.113 ms
64 bytes from 10.0.3.2: icmp_seq=1 ttl=64 time=1.114 ms
^C
--- 10.0.3.2 ping statistics ---
2 packets transmitted, 2 packets received, 0% packet loss
round-trip min/avg/max/stddev = 1.114/12.114/23.113/10.999 ms
```


#### Line Configuration

In this example I'm shutting down on of the interfaces on both `core1/core2` towards the `dst1` 

<mark style="background: #ADCCFFA6;">conf-line.yml</mark>
```
---
- name: Manipulate the configuration of Junos devices
  hosts: vr-vmx
  connection: local
  gather_facts: no
  roles:
    - Juniper.junos
  tasks:
    - name: Shutdown interface using private config mode
      juniper_junos_config:
        config_mode: 'private'
        load: 'merge'
        lines:
          - "set interfaces ge-0/0/2 disable"
      register: response
    - name: Print the config changes.
      debug:
        var: response.diff_lines
```

To run the playbook:
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-playbook ./playbooks/conf-line.yml -i ./inventory/inventory.yml --limit=CORE
[WARNING]: Invalid characters were found in group names but not replaced, use -vvvv to see details

PLAY [Manipulate the configuration of Junos devices] *************************************************************************************

TASK [Shutdown interface using private config mode] **************************************************************************************
changed: [core2]
changed: [core1]

TASK [Print the config changes.] *********************************************************************************************************
ok: [core1] => {
    "response.diff_lines": [
        "",
        "[edit interfaces ge-0/0/2]",
        "+   disable;"
    ]
}
ok: [core2] => {
    "response.diff_lines": [
        "",
        "[edit interfaces ge-0/0/2]",
        "+   disable;"
    ]
}

PLAY RECAP *******************************************************************************************************************************
core1                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
core2                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0 
```

Results:
```
anton@core2> show interfaces descriptions 
Interface       Admin Link Description
ge-0/0/0        up    up   ## Link to ISP1 (eth1)## 
ge-0/0/1        up    up   ## Link to core1 (eth2)## 
ge-0/0/2        down  down ## Link to dst1 (eth3)## 
lo0             up    up   ## System_Loopback ##
```

#### Rollback

It is important to be able to roll-back configuration or changes made to configuration when needed.

This example explores rolling back the last applied configuration change, in this case being the shutting down of the interfaces on the `core1/core2` nodes.

<mark style="background: #ADCCFFA6;">conf-rollback-1.yml</mark>
```
---
- name: Manipulate the configuration of Junos devices
  hosts: vr-vmx
  connection: local
  gather_facts: no
  roles:
    - Juniper.junos
  tasks:
    - name: Rollback to the previous config.
      juniper_junos_config:
        config_mode: 'private'
        rollback: 1
      register: response

    - name: Print the config changes.
      debug:
        var: response.diff_lines
```

Results:
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-playbook ./playbooks/conf-rollback-1.yml -i ./inventory/inventory.yml --limit=CORE

PLAY [Manipulate the configuration of Junos devices] *************************************************************************************

TASK [Rollback to the previous config.] **************************************************************************************************
changed: [core1]
changed: [core2]

TASK [Print the config changes.] *********************************************************************************************************
ok: [core1] => {
    "response.diff_lines": [
        "",
        "[edit interfaces ge-0/0/2]",
        "-   disable;"
    ]
}
ok: [core2] => {
    "response.diff_lines": [
        "",
        "[edit interfaces ge-0/0/2]",
        "-   disable;"
    ]
}

PLAY RECAP *******************************************************************************************************************************
core1                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
core2                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0 
```

### Templating Configuration
*This lab is not an example if CI/CD pipeline - but rather a demonstration of the basic principles of automating a network.

The whole idea of automation is to make your life easier in the long run. Rather than replicating similar configuration files for each individual node lets put together a template and pass the specific variables to that template to render the configuration files for each individual node.

We will use Jinja2 template for configuring eBGP between `core1/core2` and `isp1` and render it with the built in Ansible module.

<mark style="background: #ADCCFFA6;">eBGP.j2</mark>
```
set routing-options router-id {{ hostvars[inventory_hostname].ip_lo.split('/')[0] }}
set routing-options autonomous-system {{ hostvars[inventory_hostname].as_local }}
set protocols bgp group BGP-EXT type external
set protocols bgp group BGP-EXT local-address {{ hostvars[inventory_hostname].local_peer_address }}
set protocols bgp group BGP-EXT peer-as {{ hostvars[inventory_hostname].as_peer }}
set protocols bgp group BGP-EXT neighbor {{ hostvars[inventory_hostname].remote_peer_address }}
```

The host specific variables that will be used to render the template are placed into <mark style="background: #CACFD9A6;">host_vars</mark> directory.

```
anton@mcc:~/automatedNetwork/lab-01/playbooks/host_vars$ tree
.
├── core1.yml
├── core2.yml
└── dst1.yml

0 directories, 3 files
```

This is an example of these variables:
<mark style="background: #ADCCFFA6;">core1.yml</mark>
```
anton@mcc:~/automatedNetwork/lab-01/playbooks/host_vars$ cat core1.yml 

---
hostname: core1

# eBGP vars
ip_lo: 1.1.1.1/32
as_local: 65001
as_peer: 65000
local_peer_address: 10.0.1.1
remote_peer_address: 10.0.1.10
```

The following two playbooks will load the variables and then render them into the host specific configuration files:

<mark style="background: #ADCCFFA6;">render-eBGP-template.yml </mark>
```
anton@mcc:~/automatedNetwork/lab-01/playbooks$ cat render-eBGP-template.yml 
- hosts: localhost
  gather_facts: false

  tasks:
    - name: Generate configurations for all core nodes
      include_tasks: per-node-eBGP-tasks.yml
      vars:
        core: "{{ item }}"
      loop:
        - core1
        - core2
      loop_control:
        loop_var: item
```

<mark style="background: #ADCCFFA6;">per-node-eBGP-tasks.yml</mark>
```
anton@mcc:~/automatedNetwork/lab-01/playbooks$ cat per-node-eBGP-tasks.yml 
- name: Load router-specific variables
  ansible.builtin.include_vars:
    file: "host_vars/{{ core }}.yml"

- name: Render config files
  ansible.builtin.template:
    src: "/home/anton/automatedNetwork/lab-01/templates/eBGP.j2"
    dest: "/home/anton/automatedNetwork/lab-01/configuration/{{ core }}-eBGP.txt"
```

Rendered configuration files are placed into the configuration folder:

```
anton@mcc:~/automatedNetwork/lab-01/configuration$ tree
.
├── core1-eBGP.txt
├── core1.txt
├── core2-eBGP.txt
├── core2.txt
├── dst1.txt
└── sshRSA.cfg

0 directories, 6 files
```


Next stage is to load these eBGP configuration files onto the nodes. The playbook is a copy of our previous <mark style="background: #ADCCFFA6;">conf-file.yml </mark> playbook, modified for these specific files. 


<mark style="background: #ADCCFFA6;">conf-eBGP-file.yml</mark>
```
---
- name: Load and commit eBGP configuration files
  hosts: CORE
  gather_facts: false
  connection: local
  roles:
    - Juniper.junos
  tasks:
    - name: Load configuration from a local file and commit
      juniper_junos_config:
        load: merge
        format: set
        src: "/home/anton/automatedNetwork/lab-01/configuration/{{ inventory_hostname.split('-')[0] }}-eBGP.txt"
      register: response

    - name: Print the response
      debug:
        var: response
```

Lets put together one more playbook, that will first render the configuration files and then load them onto the nodes:

<mark style="background: #ADCCFFA6;">render-conf-eBGP.yml</mark>
```
anton@mcc:~/automatedNetwork/lab-01$ ansible-playbook ./playbooks/render-conf-eBGP.yml -i inventory/inventory.yml 
[WARNING]: Invalid characters were found in group names but not replaced, use -vvvv to see details

PLAY [localhost] ***************************************************************************************************

TASK [Generate configurations for all core nodes] ******************************************************************
included: /home/anton/automatedNetwork/lab-01/playbooks/per-node-eBGP-tasks.yml for localhost => (item=core1)
included: /home/anton/automatedNetwork/lab-01/playbooks/per-node-eBGP-tasks.yml for localhost => (item=core2)

TASK [Load router-specific variables] ******************************************************************************
ok: [localhost]

TASK [Render config files] *****************************************************************************************
ok: [localhost]

TASK [Load router-specific variables] ******************************************************************************
ok: [localhost]

TASK [Render config files] *****************************************************************************************
ok: [localhost]

PLAY [Load and commit eBGP configuration files] ********************************************************************

TASK [Load configuration from a local file and commit] *************************************************************
ok: [core2]
ok: [core1]

TASK [Print the response] ******************************************************************************************
ok: [core1] => {
    "response": {
        "ansible_facts": {
            "discovered_interpreter_python": "/usr/bin/python3"
        },
        "changed": false,
        "failed": false,
        "file": "/home/anton/automatedNetwork/lab-01/configuration/core1-eBGP.txt",
        "msg": "Configuration has been: opened, loaded, checked, diffed, closed."
    }
}
ok: [core2] => {
    "response": {
        "ansible_facts": {
            "discovered_interpreter_python": "/usr/bin/python3"
        },
        "changed": false,
        "failed": false,
        "file": "/home/anton/automatedNetwork/lab-01/configuration/core2-eBGP.txt",
        "msg": "Configuration has been: opened, loaded, checked, diffed, closed."
    }
}

PLAY RECAP *********************************************************************************************************
core1                      : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
core2                      : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   
localhost                  : ok=6    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   

```


Testing:

As you can see the eBGP configuration has been successfully applied  to the nodes and we have a peering with `'isp1'`.

The routes are being received but not accepted yet, as we don't have any  prefix lists or policy maps.
```
anton@core1> show bgp summary    
Threading mode: BGP I/O
Default eBGP mode: advertise - accept, receive - accept
Groups: 1 Peers: 1 Down peers: 0
Table          Tot Paths  Act Paths Suppressed    History Damp State    Pending
inet.0               
                     255          0          0          0          0          0
Peer                     AS      InPkt     OutPkt    OutQ   Flaps Last Up/Dwn State|#Active/Received/Accepted/Damped...
10.0.1.10             65000          4          2       0       3           6 Establ
  inet.0: 0/255/0/0

anton@core1> exit 

Connection to core1 closed.
anton@mcc:~/automatedNetwork/lab-01$ ssh core2
Last login: Thu Jul 27 22:58:24 2023 from 10.0.0.2
--- JUNOS 22.3R1.11 Kernel 64-bit  JNPR-12.1-20220816.a81ed05_buil
anton@core2> show bgp summary 
Threading mode: BGP I/O
Default eBGP mode: advertise - accept, receive - accept
Groups: 1 Peers: 1 Down peers: 0
Table          Tot Paths  Act Paths Suppressed    History Damp State    Pending
inet.0               
                     255          0          0          0          0          0
Peer                     AS      InPkt     OutPkt    OutQ   Flaps Last Up/Dwn State|#Active/Received/Accepted/Damped...
10.0.2.10             65000          4          2       0       1          20 Establ
  inet.0: 0/255/0/0
```

#### Outro

Although this is the very basic concept lab, hopefully it provides the necessary building blocks to develop further. By exploring information gathering, templating configuration, and deploying configuration, you now possess the essential building blocks to delve deeper into the world of network automation.

Remember, this lab is just the beginning of your journey towards harnessing the power of Ansible for managing Junos devices efficiently. With this foundation, you can now explore more advanced automation techniques, integrate Ansible with other tools, and tackle real-world network automation challenges.

Keep exploring, learning, and experimenting with Ansible and Junos OS to enhance your skills and bring greater automation and efficiency to your network management tasks. Happy automating!
