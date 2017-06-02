package edu.jhu.order.deephealth;

public class App 
{
    public static void main( String[] args )
    {
      if (args.length != 4) {
        System.out.println("Usage: HOST PORT MODULE OBSERVER");
        System.exit(1);
      }
      String host = args[0];
      int port = Integer.parseInt(args[1]);
      DHClient client = new DHClient(host, port, args[2], args[3], false);
      client.Init();
      client.SubmitReport("TS_2", 
          DHBuilder.NewMetric("cpu", Health.Status.UNHEALTHY, 30.0f),
          DHBuilder.NewMetric("memory", Health.Status.HEALTHY, 80.0f));
      client.SubmitReport("TS_2", 
          DHBuilder.NewMetric("disk", Health.Status.UNHEALTHY, 20.0f));
      client.GetLatestReport("TS_2");
      client.GetLatestReport("TS_3");
      System.out.println("Done!");
    }
}
