# cernbox-migration-database
Tool for migration shares from CERNBox 8 to CERNBox 9


```
Usage of ./migrate-shares-to-cernbox9:
  -debug
    	Shows debugging info
  -dryrun
    	Execute logic without commiting changes to the databases
  -sourcedbhost string
    	The host of the source db
  -sourcedbname string
    	The name of the source db
  -sourcedbpass string
    	The pass to connect to the source db
  -sourcedbport int
    	The port of the source db (default 3306)
  -sourcedbusername string
    	The username to connect to the source db
  -targetdbhost string
    	The host of the target db
  -targetdbname string
    	The name of the target db
  -targetdbpass string
    	The pass to connect to the target db
  -targetdbport int
    	The port of the target db (default 3306)
  -targetdbusername string
    	The username to connect to the target db
  -user string
    	Limit the migration to this user shares
```
