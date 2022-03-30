from datetime import datetime
import time


def convert_timestamp_to_datetime(timestamp):
    return datetime.fromtimestamp(timestamp).strftime('%Y-%m-%d %H:%M:%S')


def convert_datetime_to_timestamp(ds):
    obj_datetime = datetime.strptime(ds, '%Y-%m-%d %H:%M:%S')
    return int(time.mktime(obj_datetime.timetuple()))