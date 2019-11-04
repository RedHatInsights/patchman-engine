from peewee import *

database = PostgresqlDatabase("spm_db")

class Host(Model):
    id = PrimaryKeyField(),
    request = CharField()
    checksum = CharField()


