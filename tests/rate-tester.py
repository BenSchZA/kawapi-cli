import requests
import sched, time
from ratelimit import limits

scheduler = sched.scheduler(time.time, time.sleep)

def timed_call(calls_per_second, callback, *args, **kw):
    print("Timed call")
    period = 1.0 / calls_per_second
    print(period)
    def reload():
        callback(*args, **kw)
        scheduler.enter(period, 10, reload, ())
    scheduler.enter(period, 10, reload, ())

# @limits(calls=60, period=60)
def perform_request():
    response = requests.get("http://localhost:8080/tokenABCD/endpoint/a/api/project")
    print(response)

# timed_call(1, perform_request)

while True:
    perform_request()
    time.sleep(.500)