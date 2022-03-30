from concurrent import futures

import grpc
import pandas as pd

import analyze
import rightsizing_pb2
import rightsizing_pb2_grpc
import utils


class Forecast(rightsizing_pb2_grpc.Forecast):
    """Provides rightsizing analysis on request"""
    def Forecast(self, request, context):
        time_series_data = [(utils.convert_timestamp_to_datetime(ts.timestamp), ts.value) for ts in request.data]
        # timestamps, values = list(zip(*data))
        df = pd.DataFrame(time_series_data, columns=["ds", "y"])

        forecast = analyze.forecasting(df)

        response = rightsizing_pb2.ForecastResponse(id=request.id)
        result = response.result
        for data_type in ['yhat', 'yhat_upper', 'yhat_lower']:
            data = result.add()
            data.name = data_type
            analysis_data = [
                rightsizing_pb2.TimeSeriesDatapoint(timestamp=utils.convert_datetime_to_timestamp(ts), value=value)
                for ts, value in zip(forecast['ds'], forecast[data_type])]
            data.data.extend(analysis_data)
        return response


class Rightsizing(rightsizing_pb2_grpc.Rightsizing):
    def Rightsizing(self, request, context):
        optimized_usage = analyze.percentile(request.data, 95)
        response = rightsizing_pb2.RightsizingResponse(id=request.id, result=optimized_usage)
        return response


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    # Add servicer to server
    rightsizing_pb2_grpc.add_ForecastServicer_to_server(Forecast(), server)
    rightsizing_pb2_grpc.add_RightsizingServicer_to_server(Rightsizing(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
