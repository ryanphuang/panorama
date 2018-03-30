package edu.jhu.order.deephealth;

import java.util.HashMap;
import java.util.HashSet;
import java.util.logging.Logger;

public class DHPendingTracker extends Thread {

	private static final Logger logger = Logger.getLogger(DHPendingTracker.class.getName());

  int expirationInterval;

  volatile boolean running = true;
  volatile long currentTime;

  long nextExpirationTime;

  HashMap<String, DHPendingRequest> pendingRequests = new HashMap<String, DHPendingRequest>();
  HashMap<Long, HashSet<DHPendingRequest>> expiringRequests = new HashMap<Long, HashSet<DHPendingRequest>>();

  DHRequestProcessor processor;

  public static class DHPendingRequest {
    public String subject;
    public String name;
    public float score;
    public long time;    // time when the request was submitted
    public long expire;  // time to expire

    public DHPendingRequest(String subject, String name, float score, long time) {
      this.subject = subject;
      this.name = name;
      this.score = score;
      this.time = time;
    }
  }

  public DHPendingTracker(DHRequestProcessor processor) {
    this.processor = processor;
  }

  private long roundToInterval(long time) {
    // We give a one interval grace period
    return (time / expirationInterval + 1) * expirationInterval;
  }

  @Override
  public void run() {
    logger.info("DHPendingTracker started");
    try {
      while (running) {
        currentTime = System.currentTimeMillis();
        if (nextExpirationTime > currentTime) {
          this.wait(nextExpirationTime - currentTime);
          continue;
        }
        synchronized (expiringRequests) {
          HashSet<DHPendingRequest> reqSet = expiringRequests.get(nextExpirationTime);
          if (reqSet != null) {
          }
        }
        nextExpirationTime += expirationInterval;
      }
    } catch (InterruptedException e) {
      logger.info("Unexpected interruption " + e);
    }
    logger.info("DHPendingTracker exited");
  }


  synchronized public void add(String subject, String name, String id, float score) {
    long time = System.currentTimeMillis();
    DHPendingRequest req = new DHPendingRequest(subject, name, score, time);
    pendingRequests.put(id, req);
    long expireTime = roundToInterval(time + expirationInterval);
    req.expire = expireTime;
    HashSet<DHPendingRequest> reqSet = expiringRequests.get(expireTime);
    if (reqSet != null) {
      reqSet = new HashSet<DHPendingRequest>();
      expiringRequests.put(expireTime, reqSet);
    }
    reqSet.add(req);
  }

  synchronized public void clear(String subject, String name, String id, float score) {
    pendingRequests.remove(id);
  }
}
