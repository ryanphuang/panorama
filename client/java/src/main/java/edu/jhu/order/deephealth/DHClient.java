package edu.jhu.order.deephealth;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;

import java.util.concurrent.TimeUnit;
import java.util.logging.Level;
import java.util.logging.Logger;

import edu.jhu.order.deephealth.Health.Observation;
import edu.jhu.order.deephealth.Health.Report;
import edu.jhu.order.deephealth.HealthServiceGrpc.HealthServiceBlockingStub;
import edu.jhu.order.deephealth.HealthServiceGrpc.HealthServiceStub;
import edu.jhu.order.deephealth.Service.GetReportReply;
import edu.jhu.order.deephealth.Service.GetReportRequest;
import edu.jhu.order.deephealth.Service.SubmitReportReply;
import edu.jhu.order.deephealth.Service.SubmitReportRequest;

public class DHClient
{
  private static final Logger logger = Logger.getLogger(HealthServiceStub.class.getName());

  private final ManagedChannel channel;
  private final HealthServiceBlockingStub blockingStub;
  private final HealthServiceStub asyncStub;

  public DHClient(String host, int port)
  {
    this(ManagedChannelBuilder.forAddress(host, port).usePlaintext(true));
  }

  public DHClient(ManagedChannelBuilder<?> channelBuilder) 
  {
    channel = channelBuilder.build();
    blockingStub = HealthServiceGrpc.newBlockingStub(channel);
    asyncStub = HealthServiceGrpc.newStub(channel);
  }

  public void shutdown() throws InterruptedException {
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

  public SubmitReportReply.Status SubmitReport(String observer, String subject, Observation observation)
  {
    logger.info("Submitting report from " + observer + " to " + subject);
    Report report = Report.newBuilder().setObserver(observer).setSubject(subject)
      .setObservation(observation).build();
    SubmitReportRequest request = SubmitReportRequest.newBuilder().setReport(report)
      .build();
    SubmitReportReply reply; 
    reply = blockingStub.submitReport(request);
    SubmitReportReply.Status status = reply.getResult();
    logger.info("Result: " + status);
    return status;
  }

  public Report GetReport(String subject)
  {
    GetReportRequest request = GetReportRequest.newBuilder().setSubject(subject).build();
    GetReportReply reply = blockingStub.getReport(request);
    Report report = reply.getReport();
    logger.info("Result: " + report);
    return report;
  }
}
