from datetime import datetime, timedelta
from typing import List

from fbprophet import Prophet
import pandas as pd
import numpy as np


_MARGIN = 0.2


def percentile(values: List[float], quantile: int) -> float:
    q = np.percentile(values, quantile)
    return q * (1 + _MARGIN)


def forecasting(df, freq=None) -> pd.DataFrame:
    m = Prophet(interval_width=0.1)
    m.fit(df)

    if freq:
        future = m.make_future_dataframe(periods=1440, freq=freq)
    else:
        future = m.make_future_dataframe(periods=1440, freq="5min")
    forecast = m.predict(future)
    forecast['ds'] = forecast['ds'].astype(str)

    now = datetime.now()
    end_time = now + timedelta(hours=6)

    result = forecast[['ds', 'yhat', 'yhat_upper', 'yhat_lower']].query(
        f"ds >= '{now:%Y-%m-%d %H:%M}' and ds <= '{end_time:%Y-%m-%d %H:%M}'")
    return result
