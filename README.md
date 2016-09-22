regiond
=========

Region based HTTP proxy written in Go.

Install:

```
go get -u github.com/dddpaul/regiond
```

Region IDs are fetched from Oracle database :)

Table structure:
```
CREATE TABLE ip_to_region (
  ip    varchar2(20) NOT NULL
  ,region    number(10) NOT NULL
);
```
