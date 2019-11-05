# Sfync

[WIP] Salesforce Schema Sync Tool

## Install

```bash
$ curl https://install.freedom-man.com/sfync.sh | bash --
```

## Usage

```ruby
config do
  username ENV['SFDC_USERNAME']
  password ENV['SFDC_PASSWORD']
  endpoint 'login.salesforce.com'
end

object :Account do
  string :Hoge__c, length: 255
end

object :ABC__c do
  number :abc__c, scale: 10, precision: 2
end
``` 

apply schema file to Salesforce metadata
```bash
$ sfync apply -c schema.rb
```

initialize schema file
```bash
$ sfync init -c schema.rb
```

dump schema from Salesforce metadata
```bash
$ sfync dump -c schema.rb
```