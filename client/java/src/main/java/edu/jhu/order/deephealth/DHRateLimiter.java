package edu.jhu.order.deephealth;

import java.util.logging.Logger;

import edu.jhu.order.deephealth.Health.Metric;
import edu.jhu.order.deephealth.DHBuffer.AggregateValue;

public class DHRateLimiter
{
  private static final Logger logger = Logger.getLogger(DHRateLimiter.class.getName());
  private static final int CNT_THRESHOLD = 10;
  private static final int INTERVAL_SEC = 30;

  private DHBuffer buffer;

  public DHRateLimiter() {
    this.buffer = new DHBuffer();
  }

  public synchronized Metric vet(String subject, String name, Health.Status status, float score) {
    boolean report = false;
    AggregateValue val = buffer.insert(subject, name, status, score);
    long diff = val.last - val.first; 
    if (diff == 0) {
      // new report
      logger.info("Permitting new report for [" + subject + ":" + name + "]");
      report = true;
    } else if (diff > INTERVAL_SEC * 1000) {
      // repeated report
      val.first = val.last;
      val.cnt = 0;
      score = val.score;
      report = true;
      logger.info("Permitting repeated report for [" + subject + ":" + name + "] " + diff + " ms");
    } else {
      logger.fine("Report for [" + subject + ":" + name + "] too frequent");
    }
    if (report)
      return DHBuilder.NewMetric(name, status, score);
    else
      return null;
  }
}
