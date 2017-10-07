package edu.jhu.order.deephealth;

import java.util.Set;
import java.util.Map;
import java.util.HashMap;
import java.util.concurrent.ConcurrentHashMap;
import java.util.logging.Logger;

public class DHBuffer 
{
  private static final Logger logger = Logger.getLogger(DHBuffer.class.getName());

  private final Map<String, Map<AggregateKey, AggregateValue>> content = new 
        HashMap<String, Map<AggregateKey, AggregateValue>>();

  public class AggregateKey
  {
    public String name;
    public Health.Status status;

    public AggregateKey(String name, Health.Status status) {
      this.name = name;
      this.status = status;
    }

    @Override
    public int hashCode() {
      int result = 1;
      result = result * 17 + name.hashCode();
      result = result * 31 + status.hashCode();
      return result;
    }

    @Override
    public boolean equals(Object another) {
      if (this == another)
        return true;
      if (!(another instanceof AggregateKey))
        return false;
      AggregateKey r = (AggregateKey) another;
      return name.equals(r.name) && status.equals(r.status);
    }

    @Override
    public String toString() {
      return name + "-" + status;
    }
  }

  public class AggregateValue
  {
    public float score;
    public int cnt;
    public long first;
    public long last;

    public AggregateValue(float score) {
      this.score = score;
      this.cnt = 1;
    }
  }

  public class Aggregate
  {
    public String name;
    public Health.Status status;
    public float score;
    public int cnt;
    public long first;
    public long last;

    public Aggregate(String name, Health.Status status, float score) {
      this.name = name;
      this.status = status;
      this.score = score;
    }

    @Override
    public int hashCode() {
      int result = 17;
      result = result * 31 + name.hashCode();
      result = result * 31 + status.hashCode();
      return result;
    }

    @Override
    public boolean equals(Object another) {
      if (this == another)
        return true;
      if (!(another instanceof Aggregate))
        return false;
      Aggregate r = (Aggregate) another;
      return name == r.name && status == r.status && score == r.score && cnt == r.cnt && first == r.first && last == r.last;
    }
  }

  public AggregateValue insert(String subject, String name, Health.Status status, float score) {
    Map<AggregateKey, AggregateValue> aggs = content.get(subject);
    if (aggs == null) {
      logger.info("No aggregate map for " + subject);
      aggs = new HashMap<AggregateKey, AggregateValue>();
      content.put(subject, aggs);
    }
    AggregateKey key = new AggregateKey(name, status);
    AggregateValue val = new AggregateValue(score);
    AggregateValue previous = aggs.putIfAbsent(key, val);
    if (previous == null) {
      logger.fine("New aggregate value for " + subject + "/" + key);
      val.first = System.currentTimeMillis();
      val.last = val.first;
      aggs.put(key, val);
      return val;
    } else {
      logger.finest("Existing aggregate value for " + subject + "/" + key);
      previous.cnt++;
      previous.last = System.currentTimeMillis();
      //TODO: add score
      return previous;
    }
  }
}
