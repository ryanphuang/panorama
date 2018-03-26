package edu.jhu.order.deephealth;

import java.util.logging.Logger;
import java.util.EnumSet;

import edu.jhu.order.deephealth.Health.Metric;
import edu.jhu.order.deephealth.DHBuffer.AggregateValue;

public class DHRateLimiter
{
  private static final Logger logger = Logger.getLogger(DHRateLimiter.class.getName());
  private static final int CNT_THRESHOLD = 5000;
  private static final int INTERVAL_SEC = 20;

  // only buffer healthy reports
  private static final EnumSet<Health.Status> AGG_STATUS = EnumSet.of(Health.Status.HEALTHY);

  private DHBuffer buffer;

  public DHRateLimiter() {
    this.buffer = new DHBuffer();
  }

  public synchronized Metric vet(String subject, String name, Health.Status status, float score) {
    boolean report = false;
    if (AGG_STATUS.contains(status)) {
      // only aggregate if the status is specified to be aggregated
      AggregateValue val = buffer.insert(subject, name, status, score);
      long diff = val.last - val.first; 
      if (val.cnt == 1) {
        // new report
        logger.info("Permitting new report for [" + subject + ":" + name + "]");
        report = true;
      } else if (diff > INTERVAL_SEC * 1000 || val.cnt >= CNT_THRESHOLD) {
        // repeated report
        score = val.score / val.cnt;
        report = true;
        logger.info("Permitting repeated report for [" + subject + ":" + name + "] " + diff + " ms");
        // now reset aggregate value
        val.reset();
      } else {
        logger.fine("Report for [" + subject + ":" + name + "] too frequent");
        report = false;
      }
    } else {
      report = true;
    }
    if (report)
      return DHBuilder.NewMetric(name, status, score);
    else
      return null;
  }
}
