import connexion
import handler
import db
import listener
from threading import Thread
from common.logging import init_logging

init_logging()

db.Host.delete().execute()

list_thread = Thread(target=listener.main)
list_thread.start()

app = connexion.FlaskApp(__name__, specification_dir='./')
app.add_api('prototype.api.yaml')
app.run(port=8081)