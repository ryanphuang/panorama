package edu.jhu.order.deephealth;

public class App 
{
    public static void main( String[] args )
    {
      if (args.length != 2) {
        System.out.println("Usage: HOST PORT");
        System.exit(1);
      }
      String host = args[0];
      int port = Integer.parseInt(args[1]);
      DHClient client = new DHClient(host, port);
      long now = System.currentTimeMillis();
      Health.Observation observation = DHBuilder.NewObservation(now,
          DHBuilder.NewMetric("cpu", Health.Status.UNHEALTHY, 30.0f),
          DHBuilder.NewMetric("memory", Health.Status.HEALTHY, 80.0f));
      client.SubmitReport("XFE_1", "TS_2", observation);
      client.GetReport("TS_2");
      client.GetReport("TS_3");
      System.out.println("Done!");
    }
}
