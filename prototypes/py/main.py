print("Python prototype start")

import connexion
import handler
import db

app = connexion.FlaskApp(__name__, specification_dir='./')
app.add_api('prototype.api.yaml')
app.run(port=8000)