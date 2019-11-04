print("Python prototype start")

import connexion
import handler
import db
import listener

from common.logging import init_logging

init_logging()


listener.main()

app = connexion.FlaskApp(__name__, specification_dir='./')
app.add_api('prototype.api.yaml')
app.run(port=8002)