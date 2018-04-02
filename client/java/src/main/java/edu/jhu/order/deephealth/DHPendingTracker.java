package edu.jhu.order.deephealth;

import java.util.Map;
import java.util.HashMap;
import java.util.HashSet;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

import edu.jhu.order.deephealth.Health.Status;

public class DHPendingTracker {

	private static final Logger logger = Logger.getLogger(DHPendingTracker.class.getName());

  int expirationInterval;

  volatile boolean running = true;

  long nextExpirationTime;

  ConcurrentMap<String, DHPendingRequest> pendingRequests = new ConcurrentHashMap<String, DHPendingRequest>();

  ScheduledExecutorService service = Executors.newScheduledThreadPool(1);

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

  Runnable expireRunnable = new Runnable() {
    @Override
    public void run() {
      long currentTime = System.currentTimeMillis();
      for (Map.Entry<String, DHPendingRequest> entry : pendingRequests.entrySet()) {
        String reqId = entry.getKey();
        DHPendingRequest req = entry.getValue();
        if (req.time + expirationInterval < currentTime) {
          pendingRequests.remove(reqId);
          logger.info("Expiring PENDING request "+ reqId);
          processor.process(new DHRequest(req.subject, req.name, 
                Status.PENDING, req.score, req.resolve, false, req.time));
        }
      }
    }
  };

  public void start() {
    logger.info("DHPendingTracker started");
    service.scheduleWithFixedDelay(expireRunnable, expirationInterval, 
        expirationInterval, TimeUnit.MILLISECONDS);
  }

  public void shutdown() {
    logger.info("Shutting down DHPendingTracker");
    service.shutdown();
    pendingRequests.clear();
    running = false;
  }

  public void add(String subject, String name, String reqId, float score, boolean resolve) {
    long time = System.currentTimeMillis();
    DHPendingRequest req = new DHPendingRequest(subject, name, score, resolve, time);
    pendingRequests.put(reqId, req);
  }

  public void clear(String subject, String name, String reqId, float score, boolean resolve) {
    if (pendingRequests.remove(reqId) == null) {
      // It's likely that the pending request has been expired and reported to DH 
      // service. In this case, we should send a follow-up healthy report
      processor.add(subject, name, Status.HEALTHY, score, resolve, true);
    }
  }
}
