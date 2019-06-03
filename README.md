# ParallelPasswordSearching
The project focuses on parallel searching for a password in the list of already known (hacked) passwords. It is a web application developed in go language. The overall application architecture consists of three core components; Client, Server and Slaves.


 • The client is responsible for making the request, to the Server, to search the password. The client program is a Web application that contacts the server. </b>
 
 
 • The server program on receiving the request, welcomes the client, analyzes and divides the task into parts and allocates these parts to Slaves, which have already registered with the server. This multi-threaded server is the core component and listens on two different ports; one it exposes to Clients to receive jobs (password search requests). The other port is for Slaves to register themselves with the server. It is responsible for efficient job and resource management. It serves as the load-balancer and equally distributes the load amongst Slaves and manages different clients and slaves. </br>
 
 
 • The slaves are the workhorses and they register with server and are assigned the tasks. Once they register, they notify the server about the file chunks (containing passwords) they have and the server uses this info for scheduling.
