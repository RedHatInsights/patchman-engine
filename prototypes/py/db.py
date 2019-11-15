from peewee import *
from common.peewee_database import DB


class BaseModel(Model):
    """Base class for tables"""
    class Meta:
        """Base class for table metadata"""
        database = DB


class Host(BaseModel):
    id = PrimaryKeyField(),
    request = CharField()
    checksum = CharField()
    updated = DateTimeField()

    class Meta:
        table_name = "hosts"


