package edu.jhu.order.deephealth;

import java.util.concurrent.LinkedBlockingQueue;
import java.util.logging.Logger;

import edu.jhu.order.deephealth.Health.Status;
import edu.jhu.order.deephealth.Health.Metric;

public class DHRequestProcessor extends Thread {

	private static final Logger logger = Logger.getLogger(DHRequestProcessor.class.getName());

  protected LinkedBlockingQueue<DHRequest> submittedRequests = new LinkedBlockingQueue<DHRequest>();

  private DHRateLimiter rateLimiter;
  private DHResolver resolver;
  private DHClient client;

  public DHRequestProcessor(DHRateLimiter limiter, DHResolver resolver, DHClient client) {
    rateLimiter = limiter;
    this.resolver = resolver;
    this.client = client;
  }

  @Override
  public void run() {
    logger.info("DHRequestProcessor started");
    try {
      while (true) {
        DHRequest request = submittedRequests.take();
        if (DHRequest.requestOfDeath == request) {
          break;
        }
        process(request);
      }
    } catch (InterruptedException e) {
      logger.severe("Unexpected interruption: " + e);
    } catch (Exception e) {
      logger.severe("Unexpected exception" + e);
    }
    logger.info("DHRequestProcessor exited loop!");
  }

  public void add(String subject, String name, Status status, float score, boolean resolve, boolean async)
  {
    long time = System.currentTimeMillis();
    DHRequest request = new DHRequest(subject, name, status, score, resolve, async, time);
    logger.info("Queuing report from about " + subject + " at " + time);
    submittedRequests.add(request);
  }

  public void process(DHRequest request)
  {
    if (request.resolve) {
      String resolved = resolver.resolve(request.subject, DHResolver.RType.IP);
      if (resolved != null)
        request.subject = resolved;
    }
    Metric metric = rateLimiter.vet(request.subject, request.name, 
        request.status, request.score);
    if (metric != null) {
      if (request.async)
        client.reportAsync(request.time, null, request.subject, metric);
      else
        client.report(request.time, request.subject, metric);
    }
  }

  public void shutdown() {
    logger.info("Shutting down DHRequestProcessor");
    submittedRequests.clear();
    submittedRequests.add(DHRequest.requestOfDeath);
  }
}
