package edu.jhu.order.deephealth;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;

import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

import edu.jhu.order.deephealth.Health.Observation;
import edu.jhu.order.deephealth.Health.Report;
import edu.jhu.order.deephealth.Health.Metric;
import edu.jhu.order.deephealth.HealthServiceGrpc.HealthServiceBlockingStub;
import edu.jhu.order.deephealth.HealthServiceGrpc.HealthServiceStub;
import edu.jhu.order.deephealth.Service.GetReportRequest;
import edu.jhu.order.deephealth.Service.ObserveReply;
import edu.jhu.order.deephealth.Service.ObserveRequest;
import edu.jhu.order.deephealth.Service.SubmitReportReply;
import edu.jhu.order.deephealth.Service.SubmitReportRequest;
import edu.jhu.order.deephealth.Service.PingRequest;
import edu.jhu.order.deephealth.Service.PingReply;
import edu.jhu.order.deephealth.Service.Peer;

import com.google.protobuf.util.Timestamps;

public class DHClient
{
  private static final Logger logger = Logger.getLogger(HealthServiceStub.class.getName());

  private String id;
  private String serverAddr;
  private int serverPort;
  private boolean async;

  private final ManagedChannel channel;
  private final HealthServiceBlockingStub blockingStub;
  private final HealthServiceStub asyncStub;

  public DHClient(String addr, int port, String ID, boolean async)
  {
    this.serverAddr = addr;
    this.serverPort = port;
    this.id = ID;
    this.async = async;

    ManagedChannelBuilder<?> channelBuilder = ManagedChannelBuilder.forAddress(
        serverAddr, serverPort).usePlaintext(true);
    this.channel = channelBuilder.build();
    this.blockingStub = HealthServiceGrpc.newBlockingStub(channel);
    this.asyncStub = HealthServiceGrpc.newStub(channel);
  }

  public String getServerAddr()
  {
    return serverAddr;
  }

  public int getServerPort()
  {
    return serverPort;
  }

  public String getClientId()
  {
    return id;
  }

  public void setClientId(String ID)
  {
    id = ID;
  }

  public void shutdown() throws InterruptedException 
  {
    channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
  }

  public HealthServiceBlockingStub block()
  {
    return blockingStub;
  }

  public HealthServiceStub async()
  {
    return asyncStub;
  }

  public boolean Observe(String subject)
  {
    logger.info("Start observing " + subject);
    ObserveRequest request = ObserveRequest.newBuilder().setSubject(subject).build();
    ObserveReply reply;
    try {
      reply = blockingStub.observe(request);
    } catch (StatusRuntimeException e) {
      logger.warning("Observe RPC failed: " + e.getStatus());
      return false;
    }
    boolean ok = reply.getSuccess();
    logger.info("Result: " + ok);
    return ok;
  }

  public boolean StopObserving(String subject)
  {
    logger.info("Stop observing " + subject);
    ObserveRequest request = ObserveRequest.newBuilder().setSubject(subject).build();
    ObserveReply reply; 
    try {
      reply = blockingStub.stopObserving(request);
    } catch (StatusRuntimeException e) {
      logger.warning("StopObserving RPC failed: " + e.getStatus());
      return false;
    }
    boolean ok = reply.getSuccess();
    logger.info("Result: " + ok);
    return ok;
  }

  public SubmitReportReply.Status selfReport(String subject, Metric... metrics)
  {
    return SubmitReport(this.id, metrics);
  }

  public SubmitReportReply.Status SubmitReport(String subject, Metric... metrics)
  {
    long timeMillis = System.currentTimeMillis();
    logger.info("Submitting report from " + id + " about " + subject + " at " + timeMillis);
    Observation observation = DHBuilder.NewObservation(timeMillis, metrics);
    Report report = Report.newBuilder().setObserver(id).setSubject(subject)
      .setObservation(observation).build();
    SubmitReportRequest request = SubmitReportRequest.newBuilder().setReport(report)
      .build();
    SubmitReportReply reply; 
    try {
      reply = blockingStub.submitReport(request);
    } catch (StatusRuntimeException e) {
      logger.warning("SubmitReport RPC failed: " + e.getStatus());
      return null;
    }
    SubmitReportReply.Status status = reply.getResult();
    logger.info("Result: " + status);
    return status;
  }

  public Report GetLatestReport(String subject)
  {
    logger.info("Getting report for " + subject);
    GetReportRequest request = GetReportRequest.newBuilder().setSubject(subject).build();
    Report report;
    try {
      report = blockingStub.getLatestReport(request);
    } catch (StatusRuntimeException e) {
      logger.warning("GetLatestReport RPC failed: " + e.getStatus());
      return null;
    }
    logger.info("Result: " + report);
    return report;
  }

  // ping local health server
  public long Ping()
  {
    long timeMillis = System.currentTimeMillis();
    Peer source = Peer.newBuilder().setId(id).setAddr("localhost").build();
    PingRequest request = PingRequest.newBuilder().setSource(source).setTime(Timestamps.fromMillis(timeMillis)).build();
    PingReply reply;
    try {
      reply = blockingStub.ping(request);
    } catch (StatusRuntimeException e) {
      logger.warning("Ping RPC failed: " + e.getStatus());
      return -1;
    }
    long result = Timestamps.toMillis(reply.getTime());
    Date date = new Date(result);
    DateFormat formatter = new SimpleDateFormat("HH:mm:ss:SSS");
    logger.info("Got ping reply with time: " + formatter.format(date));
    return result;
  }
}
