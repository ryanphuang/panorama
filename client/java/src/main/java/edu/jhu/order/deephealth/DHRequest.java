package edu.jhu.order.deephealth;

import edu.jhu.order.deephealth.Health.Status;

public class DHRequest {
  public String subject;
  public String name;
  public Status status;
  public float score;
  public boolean resolve;
  public boolean async;
  public long time;  // time when the request was submitted

  public final static DHRequest requestOfDeath = new DHRequest(null, null, 
      Status.INVALID, 0, false, false, -1);

  public DHRequest(String subject, String name, Status status, 
      float score, boolean resolve, boolean async, long time) {
    this.subject = subject;
    this.name = name;
    this.status = status;
    this.score = score;
    this.resolve = resolve;
    this.async = async;
    this.time = time;
  }
}
