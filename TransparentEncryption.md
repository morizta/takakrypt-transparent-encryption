I want to create agent, like cte cipher trust transparent encryption expectation, encryption on the fly, for the kms or cm for config all already in another project so for this is only the agent key operation are in my kms project, setup policy, guard point resource set and user set, this agent will pull or receive the data from it, the agent can have more than one guard point root path, each have one policy, one policy can have more than one security_rules when create policy it will have one default at order 0, that cant delete and edited it is for default rules, user set and resource set are who act it (user set are for user, process set are system or apps), user set are the config for user, process set are for process config for file db access, process nano, etc. resurce set are folder, 
when set user dont need to set the proceess, when set the process dont need to set the user,
action are all operapable action like read, write and other how do i need to achieve this

so when user open a file or write a file will check the policy is it authorize or not, browse are the ls function, when apply key checked when user are not authorize it will return permission denied when it check it can be readed but return cipher text,
for process set is for the binary system or app can acceesss the file wether it is db file or another.

expected for high performance, robust clean code