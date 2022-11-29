# SUSE Manager - http rest api - golang 
Here is a golang application that queries SUSE Manager http rest api.

For testing with the source code the system needs to have go1.13 or above installed. 

__Download:__
```
git clone https://github.com/bjin01/suma-go.git

```

__Prepare and create the sumaconf.yaml with login credentials:__
e.g. 
```
server: suma.domain.xx
user: admin
password: test!
```

__Execute__

```
go run main.go -sumaconf sumaconf.yaml -schedule 3
```

sumaconfig.yaml - holds the SUMA server name and login credentials that will be parsed by the program.

-schedule - argument is prepared for the update job schedule time. with the number given the update job will be scheduled from now + given hours in the future.

For example if -schedule 3 then the job start will be 3 hours from now on.
