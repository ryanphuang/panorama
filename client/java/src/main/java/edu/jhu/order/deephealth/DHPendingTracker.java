package edu.jhu.order.deephealth;

import java.util.Map;
import java.util.HashMap;
import java.util.HashSet;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.logging.Logger;

import edu.jhu.order.deephealth.Health.Status;

public class DHPendingTracker extends Thread {

	private static final Logger logger = Logger.getLogger(DHPendingTracker.class.getName());

  int expirationInterval;

  volatile boolean running = true;
  volatile long currentTime;

  long nextExpirationTime;

  ConcurrentMap<String, DHPendingRequest> pendingRequests = new ConcurrentHashMap<String, DHPendingRequest>();

  DHRequestProcessor processor;

  public static class DHPendingRequest {
    public String subject;
    public String name;
    public float score;
    public boolean resolve;
    public long time;    // time when the request was submitted

    public DHPendingRequest(String subject, String name, float score, boolean resolve, long time) {
      this.subject = subject;
      this.name = name;
      this.score = score;
      this.resolve = resolve;
      this.time = time;
    }
  }

  public DHPendingTracker(DHRequestProcessor processor, int expirationInterval) {
    this.processor = processor;
    this.expirationInterval = expirationInterval;
  }

  public void setExpirationInterval(int interval) {
    expirationInterval = interval;
  }

  private long roundToInterval(long time) {
    return (time / expirationInterval + 1) * expirationInterval;
  }

  @Override
  public void run() {
    logger.info("DHPendingTracker started");
    nextExpirationTime = roundToInterval(System.currentTimeMillis());
    try {
      while (running) {
        currentTime = System.currentTimeMillis();
        if (nextExpirationTime > currentTime) {
          this.wait(nextExpirationTime - currentTime);
          continue;
        }
        for (Map.Entry<String, DHPendingRequest> entry : pendingRequests.entrySet()) {
          DHPendingRequest req = entry.getValue();
          if (req.time + expirationInterval < currentTime) {
            pendingRequests.remove(entry.getKey());
            processor.process(new DHRequest(req.subject, req.name, 
                  Status.PENDING, req.score, req.resolve, false, req.time));
          }
        }
        nextExpirationTime += expirationInterval;
      }
    } catch (InterruptedException e) {
      logger.info("Unexpected interruption " + e);
    }
    logger.info("DHPendingTracker exited");
  }

  public void shutdown() {
    logger.info("Shutting down DHPendingTracker");
    pendingRequests.clear();
    running = false;
  }

  public void add(String subject, String name, String id, float score, boolean resolve) {
    long time = System.currentTimeMillis();
    DHPendingRequest req = new DHPendingRequest(subject, name, score, resolve, time);
    pendingRequests.put(id, req);
  }

  public void clear(String subject, String name, String id, float score, boolean resolve) {
    pendingRequests.remove(id);
    processor.add(subject, name, Status.HEALTHY, score, resolve, true);
  }
}
