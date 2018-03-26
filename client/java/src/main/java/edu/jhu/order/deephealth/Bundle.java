package edu.jhu.order.deephealth;

import edu.jhu.order.deephealth.Health.Report;

import com.google.protobuf.Message;
import java.util.concurrent.CountDownLatch;

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
      client.reportAsync(time, new DHClient.AsyncCallBack() {
        public void onMessage(Message message) {
          System.out.println("Received reply message: " + message);
        }
        public void onRpcError(Throwable exception) {
          System.err.println("Received reply error: " + exception);
          finishLatch.countDown();
        }
        public void onCompleted() {
          finishLatch.countDown();
        }
      }, "TS_3", DHBuilder.NewMetric("cpu", Health.Status.UNHEALTHY, 40.0f), DHBuilder.NewMetric("disk", Health.Status.HEALTHY, 70.0f));
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
