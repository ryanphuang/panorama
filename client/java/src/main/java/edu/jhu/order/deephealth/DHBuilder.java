package edu.jhu.order.deephealth;

import java.util.HashMap;
import java.util.Map;

import com.google.protobuf.util.Timestamps;

public final class DHBuilder {

  public static Health.Metric NewMetric(String name, Health.Status status, float score) {
    return Health.Metric.newBuilder().setName(name).setValue(Health.Value.newBuilder().setStatus(status).setScore(score).build()).build();
  }

  public static Map<String, Health.Metric> NewMetrics(String...names) {
    Map<String, Health.Metric> metrics = new HashMap<String, Health.Metric>();
    for (String name : names) {
      metrics.put(name, NewMetric(name, Health.Status.INVALID, 0.0f));
    }
    return metrics;
  }

  public static Health.Observation NewObservation(long timeMillis, Health.Metric... metrics) {
    Health.Observation.Builder builder = Health.Observation.newBuilder().setTs(Timestamps.fromMillis(timeMillis));
    for (Health.Metric metric : metrics) {
      builder.putMetrics(metric.getName(), metric);
    }
    return builder.build();
  }
}
