package edu.jhu.order.deephealth;

import java.util.logging.Logger;

import edu.jhu.order.deephealth.DHBuffer.AggregateValue;

public class DHRateLimiter
{
  private static final Logger logger = Logger.getLogger(DHRateLimiter.class.getName());
  private static final int CNT_THRESHOLD = 10;
  private static final int INTERVAL_SEC = 30;

  private DHBuffer buffer;
  private DHClient client;

  public DHRateLimiter(DHClient client) {
    this.client = client;
    this.buffer = new DHBuffer();
  }

  public void vet(String subject, String name, Health.Status status, float score, boolean async) {
    AggregateValue val = buffer.insert(subject, name, status, score);
    if (val.cnt == 1) {
      // new report
      if (async)
        client.reportAsync(null, subject, DHBuilder.NewMetric(name, status, score));
      else
        client.report(subject, DHBuilder.NewMetric(name, status, score));
      logger.info("Permitting new report for [" + subject + ":" + name + "]");
    } else if (val.last - val.first > INTERVAL_SEC * 1000) {
      if (async)
        client.reportAsync(null, subject, DHBuilder.NewMetric(name, status, val.score));
      else
        client.report(subject, DHBuilder.NewMetric(name, status, val.score));
      val.cnt = 0;
      val.first = System.currentTimeMillis();
      logger.info("Permitting repeated report for [" + subject + ":" + name + "] " + val);
    } else {
      logger.info("Report for [" + subject + ":" + name + "] too frequent");
    }
  }
}
