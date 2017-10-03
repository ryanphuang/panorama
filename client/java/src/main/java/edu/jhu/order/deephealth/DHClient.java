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
import edu.jhu.order.deephealth.Service.RegisterRequest;
import edu.jhu.order.deephealth.Service.RegisterReply;
import edu.jhu.order.deephealth.Service.Peer;

import com.google.protobuf.util.Timestamps;

public class DHClient
{
  private static final Logger logger = Logger.getLogger(HealthServiceStub.class.getName());

  private String module;
  private String id;
  private String serverAddr;
  private int serverPort;
  private long handle;

  private ManagedChannel channel;
  private HealthServiceBlockingStub blockingStub;
  private HealthServiceStub asyncStub;
  private boolean ready = false;

  private static DHClient instance = null;

  public static DHClient getInstance() {
    if (instance == null) {
      instance = new DHClient();
    }
    return instance;
  }

  private DHClient()
  {
  }

  public boolean init(String addr, int port, String module, String id)
  {
    this.serverAddr = addr;
    this.serverPort = port;
    this.module = module;
    this.id = id;

    ManagedChannelBuilder<?> channelBuilder = ManagedChannelBuilder.forAddress(
        serverAddr, serverPort).usePlaintext(true);
    this.channel = channelBuilder.build();
    this.blockingStub = HealthServiceGrpc.newBlockingStub(channel);
    this.asyncStub = HealthServiceGrpc.newStub(channel);

    RegisterRequest request = RegisterRequest.newBuilder().setModule(module).setObserver(id).build();
    RegisterReply reply;
    try {
      reply = blockingStub.register(request);
    } catch (StatusRuntimeException e) {
      logger.warning("Register RPC failed: " + e.getStatus());
      return false;
    }
    handle = reply.getHandle();
    logger.info("Got register reply with handle " + handle);
    this.ready = true;
    return true;
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
    if (!ready)
      return;
    channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    ready = false;
  }

  public HealthServiceBlockingStub block()
  {
    return blockingStub;
  }

  public HealthServiceStub async()
  {
    return asyncStub;
  }

  public boolean observe(String subject)
  {
    if (!ready)
      return false;
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

  public boolean stopObserving(String subject)
  {
    if (!ready)
      return false;
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

  public SubmitReportReply.Status selfReport(Metric... metrics)
  {
    return report(this.id, metrics);
  }

  public SubmitReportReply.Status report(String subject, Metric... metrics)
  {
    if (!ready)
      return null;
    long timeMillis = System.currentTimeMillis();
    logger.info("Submitting report from " + id + " about " + subject + " at " + timeMillis);
    Observation observation = DHBuilder.NewObservation(timeMillis, metrics);
    Report report = Report.newBuilder().setObserver(id).setSubject(subject)
      .setObservation(observation).build();
    SubmitReportRequest request = SubmitReportRequest.newBuilder().setHandle(handle).setReport(report)
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

  public Report getReport(String subject)
  {
    if (!ready)
      return null;
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
  public long ping()
  {
    if (!ready)
      return -1;
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
