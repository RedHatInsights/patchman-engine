"""
Database to peewee mappings
"""
from peewee import Model, TextField, DateTimeField, DecimalField, ForeignKeyField, BooleanField, IntegerField
from .peewee_database import DB


class UnknownField:
    """UnknownField"""

    def __init__(self, *_, **__):
        pass


class BaseModel(Model):
    """Base class for tables"""
    class Meta:
        """Base class for table metadata"""
        database = DB


class RHAccount(BaseModel):
    """rh_account table"""
    name = TextField(null=False, unique=True)

    class Meta:
        """rh_account table metadata"""
        table_name = "rh_account"


class SystemPlatform(BaseModel):
    """system_platform table"""
    inventory_id = TextField(null=False, unique=True)
    rh_account_id = ForeignKeyField(column_name="rh_account_id", model=RHAccount, field="id")
    first_reported = DateTimeField(null=False)
    s3_url = TextField(null=True)
    vmaas_json = TextField(null=True)
    json_checksum = TextField(null=True)
    last_updated = DateTimeField(null=False)
    unchanged_since = DateTimeField(null=False)
    last_evaluation = DateTimeField(null=True)
    opt_out = BooleanField(null=False)
    cve_count_cache = IntegerField(null=False)
    last_upload = DateTimeField(null=False)

    class Meta:
        """system_platform table metadata"""
        table_name = "system_platform"


class CveImpact(BaseModel):
    """cve_impact table"""
    name = TextField(null=False, unique=True)

    class Meta:
        """cve_impact table metadata"""
        table_name = "cve_impact"


class Status(BaseModel):
    """Available status-values table"""
    name = TextField(null=False, unique=True)

    class Meta:
        """status table metadata"""
        table_name = "status"


class BusinessRisk(BaseModel):
    """Available business risk values"""
    name = TextField(null=False, unique=True)

    class Meta:
        """business_risk table metadata"""
        table_name = "business_risk"


class CveMetadata(BaseModel):
    """cve_metadata table"""
    cve = TextField(index=True, null=False, unique=True)
    cvss3_score = DecimalField(index=True, null=True)
    cvss3_metrics = TextField(null=True)
    description = TextField(null=False)
    impact_id = ForeignKeyField(column_name="impact_id", model=CveImpact, field="id")
    public_date = DateTimeField(null=False)
    modified_date = DateTimeField(null=False)
    cvss2_score = DecimalField(index=True, null=True)
    cvss2_metrics = TextField(null=True)
    redhat_url = TextField(null=True)
    secondary_url = TextField(null=True)

    class Meta:
        """cve_metadata table metadata"""
        table_name = 'cve_metadata'


class CveAccountData(BaseModel):
    """cve account data"""
    cve_id = ForeignKeyField(column_name="cve_id", model=CveMetadata, field="id")
    rh_account_id = ForeignKeyField(column_name="rh_account_id", model=RHAccount, field="id")
    business_risk = ForeignKeyField(column_name="business_risk_id", model=BusinessRisk, field="id")
    business_risk_text = TextField()
    status = ForeignKeyField(column_name="status_id", model=Status, field="id")
    status_text = TextField()
    systems_affected = IntegerField(null=False)
    systems_status_divergent = IntegerField(null=False)

    class Meta:
        """cve_account_data table metadata"""
        table_name = 'cve_account_data'
        primary_key = False


class SystemVulnerabilities(BaseModel):
    """system_vulnerabilities table"""
    system_id = ForeignKeyField(column_name="system_id", model=SystemPlatform, field="id")
    cve_id = ForeignKeyField(column_name="cve_id", model=CveMetadata, field="id")
    first_reported = DateTimeField(null=False)
    when_mitigated = DateTimeField(null=True)
    status = ForeignKeyField(column_name="status_id", model=Status, field="id")
    status_text = TextField()

    class Meta:
        """system_vulnerabilities table metadata"""
        table_name = "system_vulnerabilities"


class DbVersion(BaseModel):
    """db_version table"""
    name = TextField(null=False, primary_key=True)
    version = IntegerField(null=False)

    class Meta:
        """db_version table metadata"""
        table_name = "db_version"
