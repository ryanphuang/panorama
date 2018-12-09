package edu.jhu.order.deephealth;

import edu.jhu.order.deephealth.Health.Report;
import edu.jhu.order.deephealth.Service.SubmitReportReply;

import java.util.concurrent.CountDownLatch;

import com.google.common.util.concurrent.FutureCallback;
import com.google.protobuf.Message;
import io.grpc.Status;

public class Bundle
{
    public static void main( String[] args )
    {
      if (args.length != 4) {
        System.out.println("Usage: HOST PORT MODULE OBSERVER");
        System.exit(1);
      }
      String host = args[0];
      int port = Integer.parseInt(args[1]);
      DHClient client = DHClient.getInstance();
      client.init(host, port, args[2], args[3]);
      long time = System.currentTimeMillis();
      client.report(time, "TS_2", DHBuilder.NewMetric("cpu", Health.Status.UNHEALTHY, 30.0f), DHBuilder.NewMetric("memory", Health.Status.HEALTHY, 80.0f));
      client.report(time, "TS_2", DHBuilder.NewMetric("disk", Health.Status.UNHEALTHY, 20.0f));
      Report report = client.getReport("TS_2");
      System.out.println("Report for TS_2: " + report);
      report = client.getReport("TS_3");
      System.out.println("Report for TS_3: " + report);

      time = System.currentTimeMillis();
      final CountDownLatch finishLatch = new CountDownLatch(1);
      client.reportAsync(time, new FutureCallback<SubmitReportReply>() {
        @Override
        public void onSuccess(SubmitReportReply reply) {
          SubmitReportReply.Status status = reply.getResult();
          System.out.println("Got async submit report reply: " + status);
          finishLatch.countDown();
        }
        @Override
        public void onFailure(Throwable t) {
          Status status = Status.fromThrowable(t);
          System.err.println("Failed to async submit report: " + status.getDescription());
          finishLatch.countDown();
        }
      }, "TS_3", DHBuilder.NewMetric("cpu", Health.Status.UNHEALTHY, 40.0f), 
        DHBuilder.NewMetric("disk", Health.Status.HEALTHY, 70.0f));
      try {
        finishLatch.await();
      } catch (InterruptedException exception) {
      }

      for (int i = 0; i < 30; i++) {
        client.inform("TS_4", "network", Health.Status.HEALTHY, i + 60.0f, false);
      }
      report = client.getReport("TS_4");
      System.err.println("Received rate limited reply: " + report);
      System.out.println("Done!");
    }
}
