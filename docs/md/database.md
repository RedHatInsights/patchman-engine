# Database

## Tables
Main database tables description:
- **system_platform** - stores info about registered systems. Mainly system inventory ID column (`inventory_id`) Red Hat account (`rh_account_id`) which system belongs to, JSON string with lists of installed packages, repos, modules (`vmaas_json`) needed for requesting VMaaS when evaluating system. It also stores aggregated results from evaluation - advisories counts by its types. Records are created and updated by both `listener` and `evaluator` components.
- **advisory_metadata** - stores info about advisories (`description`, `summary`, `solution` etc.). It's synced and stored on trigger by `vmaas_sync` component. It allows to display detail information about the advisory.
- **system_advisories** - stores info about advisories evaluated for particular systems (system - advisory M-N mapping table). Contains info when system advisory was firstly reported and patched (if so). Records are created and updated by `evaluator` component. It allows to display list of advisories related to a system.
- **advisory_account_data** - stores info about all advisories detected within at least one system that belongs to a given account. So it provides overall statistics about system advisories displayed by the application.
- **package_name** - names of the packages installed on systems
- **package** - list of all packages versions, precisely all EVRAs (epoch-version-release-arch)
- **system_package** - list of packages installed on a system

## Schema
![](graphics/db_diagram.png)
